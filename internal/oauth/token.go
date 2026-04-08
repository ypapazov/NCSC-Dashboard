package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/go-jose/go-jose/v4"
)

// AccessClaims is the minimal validated access token payload.
type AccessClaims struct {
	Sub   string `json:"sub"`
	Email string `json:"email"`
	Name  string `json:"name"`
	Iss   string `json:"iss"`
	Exp   int64  `json:"exp"`
	Aud   any    `json:"aud"` // string or []string
}

// VerifyAccessToken checks the JWT signature against JWKS and validates iss/exp.
func VerifyAccessToken(ctx context.Context, compact string, issuer string, jwks *JWKS) (*AccessClaims, error) {
	tok, err := jose.ParseSigned(compact, []jose.SignatureAlgorithm{
		jose.RS256, jose.RS384, jose.RS512,
		jose.ES256, jose.ES384, jose.ES512,
		jose.PS256, jose.PS384, jose.PS512,
	})
	if err != nil {
		return nil, fmt.Errorf("parse jwt: %w", err)
	}
	if len(tok.Signatures) == 0 {
		return nil, fmt.Errorf("jwt: no signatures")
	}
	kid := tok.Signatures[0].Header.KeyID

	set, err := jwks.KeySet(ctx)
	if err != nil {
		return nil, err
	}
	keys := set.Key(kid)
	if len(keys) == 0 {
		// try all keys if kid mismatch (some providers rotate)
		keys = set.Keys
	}

	var raw []byte
	var lastErr error
	for _, k := range keys {
		raw, err = tok.Verify(k)
		if err == nil {
			break
		}
		lastErr = err
	}
	if raw == nil {
		return nil, fmt.Errorf("jwt verify: %w", lastErr)
	}

	var c AccessClaims
	if err := json.Unmarshal(raw, &c); err != nil {
		return nil, err
	}
	iss := strings.TrimSuffix(issuer, "/")
	tokIss := strings.TrimSuffix(c.Iss, "/")
	if tokIss != iss {
		return nil, fmt.Errorf("jwt: invalid iss")
	}
	now := time.Now().Unix()
	if c.Exp > 0 && now > c.Exp+60 {
		return nil, fmt.Errorf("jwt: expired")
	}
	return &c, nil
}
