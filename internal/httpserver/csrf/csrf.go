package csrf

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
)

// Token returns an HMAC-SHA256 hex token derived from the access token and server secret.
func Token(secret []byte, accessToken string) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(accessToken))
	return hex.EncodeToString(mac.Sum(nil))
}

// Valid checks the client token in constant time.
func Valid(secret []byte, accessToken, clientToken string) bool {
	if clientToken == "" || accessToken == "" {
		return false
	}
	exp := Token(secret, accessToken)
	if len(exp) != len(clientToken) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(exp), []byte(clientToken)) == 1
}
