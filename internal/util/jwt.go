package util

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// JWTIssuer and JWTAudience bind tokens to this service.
// A token issued by a different service will fail audience/issuer validation,
// preventing cross-service token substitution attacks.
const (
	JWTIssuer   = "simba-api"
	JWTAudience = "simba-web"

	// MaxTokenBytes is the upper size bound we accept for a JWT cookie value.
	// A well-formed HS256 JWT with these claims is well under 1 KB.
	// Rejecting oversized values prevents repeated SHA-256 CPU work (DoS).
	MaxTokenBytes = 2048
)

// JWTClaims holds the application-specific identity claims embedded in the JWT.
// jwt.RegisteredClaims carries exp, nbf, iat, iss, aud, jti.
type JWTClaims struct {
	UserID   int    `json:"uid"`
	Fullname string `json:"name"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// GenerateJWT signs an HS256 JWT that is RFC 7519-compliant:
//   - exp  — hard expiry (RFC 7519 §4.1.4)
//   - nbf  — token not valid before issuance time (RFC 7519 §4.1.5)
//   - iat  — issuance timestamp (RFC 7519 §4.1.6)
//   - jti  — unique token ID for audit / future revocation (RFC 7519 §4.1.7)
//   - iss  — issuer claim, validated on parse (RFC 7519 §4.1.1)
//   - aud  — audience claim, validated on parse (RFC 7519 §4.1.3)
func GenerateJWT(secret []byte, userID int, fullname, role string, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := JWTClaims{
		UserID:   userID,
		Fullname: fullname,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			ID:        uuid.New().String(),
			Issuer:    JWTIssuer,
			Audience:  jwt.ClaimStrings{JWTAudience},
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}

// ParseJWT validates the JWT signature, algorithm, issuer, audience, nbf, and
// mandatory expiry. Any deviation returns an opaque error — never expose the
// reason to the caller so attackers cannot distinguish claim failures.
func ParseJWT(secret []byte, tokenString string) (*JWTClaims, error) {
	if len(tokenString) > MaxTokenBytes {
		return nil, errors.New("token exceeds maximum size")
	}

	claims := &JWTClaims{}
	token, err := jwt.ParseWithClaims(
		tokenString,
		claims,
		func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("unexpected signing method")
			}
			return secret, nil
		},
		jwt.WithIssuer(JWTIssuer),
		jwt.WithAudience(JWTAudience),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
	)
	if err != nil || !token.Valid {
		return nil, errors.New("invalid or expired token")
	}
	return claims, nil
}
