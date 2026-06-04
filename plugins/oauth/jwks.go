package oauth

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"time"
)

// VerifyJWTOptions constrains JWT claim validation during VerifyRS256JWT.
type VerifyJWTOptions struct {
	Issuers   []string // accepted "iss" values (any match); empty → skip
	Audiences []string // accepted "aud" values (any match); empty → skip
	Nonce     string   // required "nonce" claim; empty → skip
	MaxAge    time.Duration
}

// VerifyRS256JWT verifies a JWT's RS256 signature against a JWKS endpoint and
// validates its standard claims, returning the decoded claims on success. Pure
// stdlib (no JWT library) — used for the client-supplied id_token sign-in path.
func VerifyRS256JWT(ctx context.Context, token, jwksURL string, opts VerifyJWTOptions) (map[string]any, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, ErrIDTokenInvalid
	}

	var header struct {
		Kid string `json:"kid"`
		Alg string `json:"alg"`
	}
	if h, err := base64.RawURLEncoding.DecodeString(parts[0]); err != nil {
		return nil, ErrIDTokenInvalid
	} else if err := json.Unmarshal(h, &header); err != nil {
		return nil, ErrIDTokenInvalid
	}
	if header.Alg != "RS256" || header.Kid == "" {
		return nil, fmt.Errorf("%w: unsupported alg/kid", ErrIDTokenInvalid)
	}

	pub, err := fetchRSAPublicKey(ctx, jwksURL, header.Kid)
	if err != nil {
		return nil, err
	}

	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, ErrIDTokenInvalid
	}
	signingInput := parts[0] + "." + parts[1]
	digest := sha256.Sum256([]byte(signingInput))
	if err := rsa.VerifyPKCS1v15(pub, crypto.SHA256, digest[:], sig); err != nil {
		return nil, fmt.Errorf("%w: signature", ErrIDTokenInvalid)
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, ErrIDTokenInvalid
	}
	var claims map[string]any
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, ErrIDTokenInvalid
	}

	if err := validateClaims(claims, opts); err != nil {
		return nil, err
	}
	return claims, nil
}

func validateClaims(claims map[string]any, opts VerifyJWTOptions) error {
	now := time.Now().UTC()
	if exp, ok := asSeconds(claims["exp"]); ok {
		if now.After(time.Unix(exp, 0)) {
			return fmt.Errorf("%w: expired", ErrIDTokenInvalid)
		}
	} else {
		return fmt.Errorf("%w: missing exp", ErrIDTokenInvalid)
	}
	if opts.MaxAge > 0 {
		if iat, ok := asSeconds(claims["iat"]); ok {
			if now.Sub(time.Unix(iat, 0)) > opts.MaxAge {
				return fmt.Errorf("%w: too old", ErrIDTokenInvalid)
			}
		}
	}
	if len(opts.Issuers) > 0 {
		iss, _ := claims["iss"].(string)
		if !contains(opts.Issuers, iss) {
			return fmt.Errorf("%w: issuer", ErrIDTokenInvalid)
		}
	}
	if len(opts.Audiences) > 0 {
		aud, _ := claims["aud"].(string)
		if !contains(opts.Audiences, aud) {
			return fmt.Errorf("%w: audience", ErrIDTokenInvalid)
		}
	}
	if opts.Nonce != "" {
		nonce, _ := claims["nonce"].(string)
		if nonce != opts.Nonce {
			return fmt.Errorf("%w: nonce", ErrIDTokenInvalid)
		}
	}
	return nil
}

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

func fetchRSAPublicKey(ctx context.Context, jwksURL, kid string) (*rsa.PublicKey, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, jwksURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var jwks struct {
		Keys []struct {
			Kid string `json:"kid"`
			Kty string `json:"kty"`
			N   string `json:"n"`
			E   string `json:"e"`
		} `json:"keys"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return nil, fmt.Errorf("oauth: decode jwks: %w", err)
	}
	for _, k := range jwks.Keys {
		if k.Kid != kid {
			continue
		}
		if k.Kty != "RSA" {
			return nil, fmt.Errorf("%w: unsupported key type %q", ErrIDTokenInvalid, k.Kty)
		}
		nBytes, err := base64.RawURLEncoding.DecodeString(k.N)
		if err != nil {
			return nil, ErrIDTokenInvalid
		}
		eBytes, err := base64.RawURLEncoding.DecodeString(k.E)
		if err != nil {
			return nil, ErrIDTokenInvalid
		}
		return &rsa.PublicKey{
			N: new(big.Int).SetBytes(nBytes),
			E: int(bigEndianUint(eBytes)),
		}, nil
	}
	return nil, fmt.Errorf("%w: signing key %q not found", ErrIDTokenInvalid, kid)
}

// bigEndianUint reads up to 8 big-endian bytes (JWK exponent) into a uint64.
func bigEndianUint(b []byte) uint64 {
	buf := make([]byte, 8)
	copy(buf[8-len(b):], b)
	return binary.BigEndian.Uint64(buf)
}
