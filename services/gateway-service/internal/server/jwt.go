package server

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// jwtClaims are the claims we extract from the token.
type jwtClaims struct {
	UserID  string `json:"uid"`
	OrgID   string `json:"org"`
	Role    string `json:"role"`
	IsAdmin bool   `json:"adm"`
	Subject string `json:"sub"`
	Exp     int64  `json:"exp"`
}

// parseJWT validates an HS256 JWT and returns claims.
// Returns nil if the token is invalid or expired.
func parseJWT(tokenString, secret string) (*jwtClaims, error) {
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid token format")
	}

	// Verify signature (HS256)
	signingInput := parts[0] + "." + parts[1]
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signingInput))
	expectedSig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(parts[2]), []byte(expectedSig)) {
		return nil, fmt.Errorf("invalid signature")
	}

	// Decode payload
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid payload encoding")
	}

	var claims jwtClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, fmt.Errorf("invalid payload JSON")
	}

	// Check expiration
	if claims.Exp > 0 && time.Now().Unix() > claims.Exp {
		return nil, fmt.Errorf("token expired")
	}

	return &claims, nil
}

// withJWTAuth validates Bearer tokens and injects user context headers.
// If the token is valid, it sets X-User-ID, X-Org-ID, X-User-Role, X-Is-Admin headers
// for downstream services to use for data isolation.
func withJWTAuth(next http.Handler, jwtSecret string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip auth for public paths
		if isPublicPath(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		// Extract bearer token
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			writeAPIError(w, http.StatusUnauthorized, "AUTH_REQUIRED", "Authorization header required")
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			writeAPIError(w, http.StatusUnauthorized, "INVALID_AUTH_FORMAT", "expected 'Bearer <token>'")
			return
		}

		token := strings.TrimSpace(parts[1])
		if token == "" {
			writeAPIError(w, http.StatusUnauthorized, "EMPTY_TOKEN", "token is empty")
			return
		}

		claims, err := parseJWT(token, jwtSecret)
		if err != nil {
			writeAPIError(w, http.StatusUnauthorized, "INVALID_TOKEN", err.Error())
			return
		}

		// Inject identity headers for downstream services
		r.Header.Set("X-User-ID", claims.UserID)
		r.Header.Set("X-Org-ID", claims.OrgID)
		r.Header.Set("X-User-Role", claims.Role)
		if claims.IsAdmin {
			r.Header.Set("X-Is-Admin", "true")
		} else {
			r.Header.Set("X-Is-Admin", "false")
		}

		// Enforce admin-only paths
		if isAdminPath(r.URL.Path) && !claims.IsAdmin {
			writeAPIError(w, http.StatusForbidden, "ADMIN_REQUIRED", "this endpoint requires platform admin privileges")
			return
		}

		next.ServeHTTP(w, r)
	})
}

// isAdminPath returns true for paths that require platform admin privileges.
func isAdminPath(path string) bool {
	return strings.HasPrefix(path, "/api/v1/admin/") || strings.HasPrefix(path, "/api/admin/")
}
