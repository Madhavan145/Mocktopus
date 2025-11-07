package server

import (
	"net/http"
	"strings"
)

// withAPIKeyAuth validates the X-API-Key header on protected routes.
// For now, it validates format and passes the key downstream for further verification.
// In production, this would call the admin-service to validate the key.
func withAPIKeyAuth(next http.Handler, adminServiceURL string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Paths that don't require API key authentication
		if isPublicPath(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		// Check for API key
		apiKey := r.Header.Get("X-API-Key")
		authHeader := r.Header.Get("Authorization")

		// Allow either API key or Bearer token
		if apiKey == "" && authHeader == "" {
			writeAPIError(w, http.StatusUnauthorized, "AUTH_REQUIRED", "API key (X-API-Key) or Authorization header required")
			return
		}

		// If API key provided, validate format (mk_ prefix, minimum length)
		if apiKey != "" {
			if !isValidAPIKeyFormat(apiKey) {
				writeAPIError(w, http.StatusUnauthorized, "INVALID_API_KEY", "invalid API key format")
				return
			}
			// Pass the key prefix downstream for the admin-service to validate
			r.Header.Set("X-API-Key-Validated", "format-ok")
		}

		next.ServeHTTP(w, r)
	})
}

func isPublicPath(path string) bool {
	publicPaths := []string{
		"/health",
		"/metrics",
		"/api/auth/",
		"/api/v1/auth/",
		"/ws/",
	}
	for _, p := range publicPaths {
		if strings.HasPrefix(path, p) || path == p {
			return true
		}
	}
	return path == "/health" || path == "/metrics"
}

func isValidAPIKeyFormat(key string) bool {
	// API keys should be: mk_<base64 string> with minimum 20 chars total
	if len(key) < 20 {
		return false
	}
	if !strings.HasPrefix(key, "mk_") {
		return false
	}
	return true
}

// withRequestSizeLimit rejects request bodies larger than maxBytes.
func withRequestSizeLimit(next http.Handler, maxBytes int64) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ContentLength > maxBytes {
			writeAPIError(w, http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE", "request body exceeds maximum size")
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
		next.ServeHTTP(w, r)
	})
}

// withSecurityHeaders adds standard security headers to all responses.
func withSecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		next.ServeHTTP(w, r)
	})
}
