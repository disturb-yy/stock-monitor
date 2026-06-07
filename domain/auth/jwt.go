package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

const BearerPrefix = "Bearer "

var (
	ErrMissingToken       = errors.New("missing bearer token")
	ErrInvalidTokenFormat = errors.New("invalid token format")
	ErrUnsupportedAlg     = errors.New("unsupported signing algorithm")
	ErrInvalidSignature   = errors.New("invalid token signature")
	ErrExpiredToken       = errors.New("expired token")
	ErrTokenNotValidYet   = errors.New("token not valid yet")
	ErrInvalidIssuer      = errors.New("invalid issuer")
	ErrInvalidAudience    = errors.New("invalid audience")
	ErrEmptySigningSecret = errors.New("empty signing secret")
	ErrInvalidTokenPayload = errors.New("invalid token payload")
	ErrReservedClaim      = errors.New("reserved claim cannot be overwritten")
)

type JWTOptions struct {
	Secret   string
	Issuer   string
	Audience string
}

type JWTClaims struct {
	Subject   string         `json:"sub,omitempty"`
	Issuer    string         `json:"iss,omitempty"`
	Audience  any            `json:"aud,omitempty"`
	ExpiresAt float64        `json:"exp,omitempty"`
	NotBefore float64        `json:"nbf,omitempty"`
	IssuedAt  float64        `json:"iat,omitempty"`
	Extra     map[string]any `json:"-"`
}

type jwtHeader struct {
	Algorithm string `json:"alg"`
	Type      string `json:"typ"`
}

func ValidateBearerToken(headerValue string, options JWTOptions) (*JWTClaims, error) {
	return ValidateBearerTokenAt(headerValue, options, time.Now())
}

func ValidateBearerTokenAt(headerValue string, options JWTOptions, now time.Time) (*JWTClaims, error) {
	if options.Secret == "" {
		return nil, ErrEmptySigningSecret
	}

	if !strings.HasPrefix(headerValue, BearerPrefix) {
		return nil, ErrMissingToken
	}

	token := strings.TrimSpace(strings.TrimPrefix(headerValue, BearerPrefix))
	return ValidateJWTAt(token, options, now)
}

func ValidateJWT(token string, options JWTOptions) (*JWTClaims, error) {
	return ValidateJWTAt(token, options, time.Now())
}

func ValidateJWTAt(token string, options JWTOptions, now time.Time) (*JWTClaims, error) {
	if options.Secret == "" {
		return nil, ErrEmptySigningSecret
	}

	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, ErrInvalidTokenFormat
	}

	var header jwtHeader
	if err := decodeJWTPart(parts[0], &header); err != nil {
		return nil, ErrInvalidTokenFormat
	}
	if header.Algorithm != "HS256" {
		return nil, ErrUnsupportedAlg
	}

	if !validHMACSHA256(parts[0]+"."+parts[1], parts[2], options.Secret) {
		return nil, ErrInvalidSignature
	}

	claims, err := decodeJWTClaims(parts[1])
	if err != nil {
		return nil, ErrInvalidTokenPayload
	}

	if claims.ExpiresAt > 0 && now.Unix() >= int64(claims.ExpiresAt) {
		return nil, ErrExpiredToken
	}
	if claims.NotBefore > 0 && now.Unix() < int64(claims.NotBefore) {
		return nil, ErrTokenNotValidYet
	}
	if options.Issuer != "" && claims.Issuer != options.Issuer {
		return nil, ErrInvalidIssuer
	}
	if options.Audience != "" && !claims.HasAudience(options.Audience) {
		return nil, ErrInvalidAudience
	}

	return claims, nil
}

func GenerateJWT(subject string, extra map[string]any, config Config) (string, error) {
	return GenerateJWTAt(subject, extra, config, time.Now())
}

func GenerateJWTAt(subject string, extra map[string]any, config Config, now time.Time) (string, error) {
	if config.Secret == "" {
		return "", ErrEmptySigningSecret
	}

	header := map[string]any{
		"alg": "HS256",
		"typ": "JWT",
	}
	claims := map[string]any{
		"sub": subject,
		"iat": now.Unix(),
		"nbf": now.Unix(),
	}

	if config.Issuer != "" {
		claims["iss"] = config.Issuer
	}
	if config.Audience != "" {
		claims["aud"] = config.Audience
	}
	if config.TokenTTL > 0 {
		claims["exp"] = now.Add(config.TokenTTL).Unix()
	}
	for key, value := range extra {
		if isReservedClaim(key) {
			return "", ErrReservedClaim
		}
		claims[key] = value
	}

	encodedHeader, err := encodeJWTPart(header)
	if err != nil {
		return "", err
	}
	encodedClaims, err := encodeJWTPart(claims)
	if err != nil {
		return "", err
	}

	signingInput := encodedHeader + "." + encodedClaims
	signature := signHMACSHA256(signingInput, config.Secret)

	return signingInput + "." + signature, nil
}

func encodeJWTPart(value any) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(data), nil
}

func decodeJWTPart(part string, target any) error {
	data, err := base64.RawURLEncoding.DecodeString(part)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}

func decodeJWTClaims(part string) (*JWTClaims, error) {
	data, err := base64.RawURLEncoding.DecodeString(part)
	if err != nil {
		return nil, err
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	claims := &JWTClaims{Extra: map[string]any{}}
	for key, value := range raw {
		switch key {
		case "sub":
			claims.Subject, _ = value.(string)
		case "iss":
			claims.Issuer, _ = value.(string)
		case "aud":
			claims.Audience = value
		case "exp":
			claims.ExpiresAt, _ = value.(float64)
		case "nbf":
			claims.NotBefore, _ = value.(float64)
		case "iat":
			claims.IssuedAt, _ = value.(float64)
		default:
			claims.Extra[key] = value
		}
	}

	return claims, nil
}

func validHMACSHA256(signingInput, encodedSignature, secret string) bool {
	expectedSignature := signHMACSHA256Bytes(signingInput, secret)

	actualSignature, err := base64.RawURLEncoding.DecodeString(encodedSignature)
	if err != nil {
		return false
	}

	return hmac.Equal(actualSignature, expectedSignature)
}

func signHMACSHA256(signingInput, secret string) string {
	return base64.RawURLEncoding.EncodeToString(signHMACSHA256Bytes(signingInput, secret))
}

func signHMACSHA256Bytes(signingInput, secret string) []byte {
	expectedMAC := hmac.New(sha256.New, []byte(secret))
	expectedMAC.Write([]byte(signingInput))
	return expectedMAC.Sum(nil)
}

func isReservedClaim(key string) bool {
	switch key {
	case "sub", "iss", "aud", "exp", "nbf", "iat":
		return true
	default:
		return false
	}
}

func (c *JWTClaims) HasAudience(expected string) bool {
	switch audience := c.Audience.(type) {
	case string:
		return audience == expected
	case []any:
		for _, item := range audience {
			if value, ok := item.(string); ok && value == expected {
				return true
			}
		}
	}
	return false
}
