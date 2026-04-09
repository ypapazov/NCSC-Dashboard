package clamav

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"strings"
	"time"
)

type ScanResult struct {
	Clean   bool
	Virus   string
	RawLine string
}

type Client struct {
	Address string // host:port or unix socket path
	Network string // "tcp" or "unix"
	Timeout time.Duration
}

func NewTCPClient(addr string) *Client {
	return &Client{Address: addr, Network: "tcp", Timeout: 30 * time.Second}
}

func NewUnixClient(path string) *Client {
	return &Client{Address: path, Network: "unix", Timeout: 30 * time.Second}
}

// Ping sends an IDSESSION/PING command to verify connectivity.
func (c *Client) Ping(ctx context.Context) error {
	conn, err := c.dial(ctx)
	if err != nil {
		return fmt.Errorf("clamav connect: %w", err)
	}
	defer conn.Close()

	if _, err := conn.Write([]byte("zPING\x00")); err != nil {
		return fmt.Errorf("clamav ping write: %w", err)
	}

	scanner := bufio.NewScanner(conn)
	if scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "PONG") {
			return nil
		}
		return fmt.Errorf("clamav unexpected ping response: %s", line)
	}
	return fmt.Errorf("clamav: no response to ping")
}

// ScanStream sends file bytes to ClamAV for scanning via INSTREAM.
func (c *Client) ScanStream(ctx context.Context, r io.Reader) (*ScanResult, error) {
	conn, err := c.dial(ctx)
	if err != nil {
		return nil, fmt.Errorf("clamav connect: %w", err)
	}
	defer conn.Close()

	if _, err := conn.Write([]byte("zINSTREAM\x00")); err != nil {
		return nil, fmt.Errorf("clamav write command: %w", err)
	}

	buf := make([]byte, 8192)
	for {
		n, readErr := r.Read(buf)
		if n > 0 {
			// Send chunk size (4 bytes big-endian) then data.
			size := []byte{byte(n >> 24), byte(n >> 16), byte(n >> 8), byte(n)}
			if _, err := conn.Write(size); err != nil {
				return nil, fmt.Errorf("clamav write chunk size: %w", err)
			}
			if _, err := conn.Write(buf[:n]); err != nil {
				return nil, fmt.Errorf("clamav write chunk: %w", err)
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return nil, fmt.Errorf("clamav read input: %w", readErr)
		}
	}

	// Send zero-length terminator.
	if _, err := conn.Write([]byte{0, 0, 0, 0}); err != nil {
		return nil, fmt.Errorf("clamav write terminator: %w", err)
	}

	scanner := bufio.NewScanner(conn)
	if scanner.Scan() {
		return parseResponse(scanner.Text()), nil
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("clamav read response: %w", err)
	}
	return nil, fmt.Errorf("clamav: no scan response")
}

func (c *Client) dial(ctx context.Context) (net.Conn, error) {
	d := net.Dialer{Timeout: c.Timeout}
	return d.DialContext(ctx, c.Network, c.Address)
}

func parseResponse(line string) *ScanResult {
	line = strings.TrimSpace(strings.TrimRight(line, "\x00"))
	res := &ScanResult{RawLine: line}

	if strings.HasSuffix(line, "OK") {
		res.Clean = true
		return res
	}

	if idx := strings.Index(line, "FOUND"); idx > 0 {
		res.Clean = false
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			res.Virus = strings.TrimSpace(strings.TrimSuffix(parts[1], "FOUND"))
		}
		return res
	}

	res.Clean = false
	return res
}
