package server

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"runtime"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"

	"moktopus/gateway-service/internal/config"
)

var websocketUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

var websocketDialer = websocket.DefaultDialer

func New(cfg config.Config) *http.Server {
	mux := http.NewServeMux()
	startAt := time.Now()
	var requestsTotal uint64

	authTarget, err := url.Parse(cfg.AuthServiceURL)
	if err != nil {
		panic(err)
	}
	authProxy := httputil.NewSingleHostReverseProxy(authTarget)

	projectTarget, err := url.Parse(cfg.ProjectURL)
	if err != nil {
		panic(err)
	}
	projectProxy := httputil.NewSingleHostReverseProxy(projectTarget)

	topologyTarget, err := url.Parse(cfg.TopologyURL)
	if err != nil {
		panic(err)
	}
	topologyProxy := httputil.NewSingleHostReverseProxy(topologyTarget)

	simulationTarget, err := url.Parse(cfg.SimulationURL)
	if err != nil {
		panic(err)
	}
	simulationProxy := httputil.NewSingleHostReverseProxy(simulationTarget)
	simulationProxy.Transport = &http.Transport{
		ResponseHeaderTimeout: 300 * time.Second, // simulations can run up to 5 min
		IdleConnTimeout:       120 * time.Second,
	}

	metricsTarget, err := url.Parse(cfg.MetricsURL)
	if err != nil {
		panic(err)
	}
	metricsProxy := httputil.NewSingleHostReverseProxy(metricsTarget)

	costTarget, err := url.Parse(cfg.CostURL)
	if err != nil {
		panic(err)
	}
	costProxy := httputil.NewSingleHostReverseProxy(costTarget)

	chaosTarget, err := url.Parse(cfg.ChaosURL)
	if err != nil {
		panic(err)
	}
	chaosProxy := httputil.NewSingleHostReverseProxy(chaosTarget)

	replayTarget, err := url.Parse(cfg.ReplayURL)
	if err != nil {
		panic(err)
	}
	replayProxy := httputil.NewSingleHostReverseProxy(replayTarget)

	recommendationTarget, err := url.Parse(cfg.RecommendationURL)
	if err != nil {
		panic(err)
	}
	recommendationProxy := httputil.NewSingleHostReverseProxy(recommendationTarget)

	catalogTarget, err := url.Parse(cfg.CatalogURL)
	if err != nil {
		panic(err)
	}
	catalogProxy := httputil.NewSingleHostReverseProxy(catalogTarget)

	iacTarget, err := url.Parse(cfg.IaCURL)
	if err != nil {
		panic(err)
	}
	iacProxy := httputil.NewSingleHostReverseProxy(iacTarget)

	templateTarget, err := url.Parse(cfg.TemplateURL)
	if err != nil {
		panic(err)
	}
	templateProxy := httputil.NewSingleHostReverseProxy(templateTarget)

	adminTarget, err := url.Parse(cfg.AdminURL)
	if err != nil {
		panic(err)
	}
	adminProxy := httputil.NewSingleHostReverseProxy(adminTarget)

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"service":"gateway-service","status":"ok"}`)
	})

	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		uptime := time.Since(startAt).Seconds()
		requests := atomic.LoadUint64(&requestsTotal)
		goroutines := runtime.NumGoroutine()

		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		_, _ = fmt.Fprintf(w, "# HELP moktopus_gateway_http_requests_total Total HTTP requests handled by gateway\n")
		_, _ = fmt.Fprintf(w, "# TYPE moktopus_gateway_http_requests_total counter\n")
		_, _ = fmt.Fprintf(w, "moktopus_gateway_http_requests_total %d\n", requests)
		_, _ = fmt.Fprintf(w, "# HELP moktopus_gateway_uptime_seconds Uptime in seconds\n")
		_, _ = fmt.Fprintf(w, "# TYPE moktopus_gateway_uptime_seconds gauge\n")
		_, _ = fmt.Fprintf(w, "moktopus_gateway_uptime_seconds %.0f\n", uptime)
		_, _ = fmt.Fprintf(w, "# HELP moktopus_gateway_go_goroutines Number of goroutines in gateway\n")
		_, _ = fmt.Fprintf(w, "# TYPE moktopus_gateway_go_goroutines gauge\n")
		_, _ = fmt.Fprintf(w, "moktopus_gateway_go_goroutines %d\n", goroutines)
	})

	mux.HandleFunc("/api/v1/system/status", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeAPIError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
			return
		}

		writeAPIData(w, http.StatusOK, map[string]string{
			"service": "gateway-service",
			"status":  "ok",
		})
	})

	mux.HandleFunc("/api/auth/", func(w http.ResponseWriter, r *http.Request) {
		authProxy.ServeHTTP(w, r)
	})

	mux.HandleFunc("/api/topology/", func(w http.ResponseWriter, r *http.Request) {
		topologyProxy.ServeHTTP(w, r)
	})

	mux.HandleFunc("/api/projects", func(w http.ResponseWriter, r *http.Request) {
		projectProxy.ServeHTTP(w, r)
	})

	mux.HandleFunc("/api/projects/", func(w http.ResponseWriter, r *http.Request) {
		projectProxy.ServeHTTP(w, r)
	})

	mux.HandleFunc("/api/simulation/", func(w http.ResponseWriter, r *http.Request) {
		simulationProxy.ServeHTTP(w, r)
	})

	mux.HandleFunc("/api/metrics/", func(w http.ResponseWriter, r *http.Request) {
		metricsProxy.ServeHTTP(w, r)
	})

	mux.HandleFunc("/api/cost/", func(w http.ResponseWriter, r *http.Request) {
		costProxy.ServeHTTP(w, r)
	})

	mux.HandleFunc("/api/chaos/", func(w http.ResponseWriter, r *http.Request) {
		chaosProxy.ServeHTTP(w, r)
	})

	mux.HandleFunc("/api/replay/", func(w http.ResponseWriter, r *http.Request) {
		replayProxy.ServeHTTP(w, r)
	})

	mux.HandleFunc("/api/recommendation/", func(w http.ResponseWriter, r *http.Request) {
		recommendationProxy.ServeHTTP(w, r)
	})

	mux.HandleFunc("/api/catalog/", func(w http.ResponseWriter, r *http.Request) {
		catalogProxy.ServeHTTP(w, r)
	})

	mux.HandleFunc("/api/iac/", func(w http.ResponseWriter, r *http.Request) {
		iacProxy.ServeHTTP(w, r)
	})

	mux.HandleFunc("/api/templates/", func(w http.ResponseWriter, r *http.Request) {
		templateProxy.ServeHTTP(w, r)
	})

	mux.HandleFunc("/api/templates", func(w http.ResponseWriter, r *http.Request) {
		templateProxy.ServeHTTP(w, r)
	})

	mux.HandleFunc("/api/v1/admin/", func(w http.ResponseWriter, r *http.Request) {
		adminProxy.ServeHTTP(w, r)
	})

	mux.HandleFunc("/api/v1/", func(w http.ResponseWriter, r *http.Request) {
		r2 := r.Clone(r.Context())
		r2.URL.Path = strings.TrimPrefix(r.URL.Path, "/api/v1")
		if r2.URL.Path == "" {
			r2.URL.Path = "/"
		}

		switch {
		case strings.HasPrefix(r2.URL.Path, "/auth/"):
			r2.URL.Path = "/api/v1" + r2.URL.Path
			authProxy.ServeHTTP(w, r2)
		case strings.HasPrefix(r2.URL.Path, "/org"):
			r2.URL.Path = "/api/v1" + r2.URL.Path
			authProxy.ServeHTTP(w, r2)
		case strings.HasPrefix(r2.URL.Path, "/topology/"):
			r2.URL.Path = "/api/v1" + r2.URL.Path
			topologyProxy.ServeHTTP(w, r2)
		case r2.URL.Path == "/projects" || strings.HasPrefix(r2.URL.Path, "/projects/"):
			r2.URL.Path = "/api/v1" + r2.URL.Path
			projectProxy.ServeHTTP(w, r2)
		case strings.HasPrefix(r2.URL.Path, "/team/"):
			r2.URL.Path = "/api/v1" + r2.URL.Path
			projectProxy.ServeHTTP(w, r2)
		case strings.HasPrefix(r2.URL.Path, "/simulation/"):
			r2.URL.Path = "/api/v1" + r2.URL.Path
			simulationProxy.ServeHTTP(w, r2)
		case strings.HasPrefix(r2.URL.Path, "/metrics/"):
			r2.URL.Path = "/api/v1" + r2.URL.Path
			metricsProxy.ServeHTTP(w, r2)
		case strings.HasPrefix(r2.URL.Path, "/cost/"):
			r2.URL.Path = "/api/v1" + r2.URL.Path
			costProxy.ServeHTTP(w, r2)
		case strings.HasPrefix(r2.URL.Path, "/chaos/"):
			r2.URL.Path = "/api/v1" + r2.URL.Path
			chaosProxy.ServeHTTP(w, r2)
		case strings.HasPrefix(r2.URL.Path, "/replay/"):
			r2.URL.Path = "/api/v1" + r2.URL.Path
			replayProxy.ServeHTTP(w, r2)
		case strings.HasPrefix(r2.URL.Path, "/recommendation/"):
			r2.URL.Path = "/api/v1" + r2.URL.Path
			recommendationProxy.ServeHTTP(w, r2)
		case strings.HasPrefix(r2.URL.Path, "/catalog/"):
			r2.URL.Path = "/api/v1" + r2.URL.Path
			catalogProxy.ServeHTTP(w, r2)
		case strings.HasPrefix(r2.URL.Path, "/iac/"):
			r2.URL.Path = "/api/v1" + r2.URL.Path
			iacProxy.ServeHTTP(w, r2)
		case r2.URL.Path == "/templates" || strings.HasPrefix(r2.URL.Path, "/templates/"):
			r2.URL.Path = "/api/v1" + r2.URL.Path
			templateProxy.ServeHTTP(w, r2)
		default:
			writeAPIError(w, http.StatusNotFound, "ROUTE_NOT_FOUND", "v1 route not found")
		}
	})

	mux.HandleFunc("/ws/simulation_updates", func(w http.ResponseWriter, r *http.Request) {
		proxyWebSocket(w, r, cfg.SimulationURL, "/api/simulation/stream")
	})

	mux.HandleFunc("/ws/metrics_updates", func(w http.ResponseWriter, r *http.Request) {
		proxyWebSocket(w, r, cfg.MetricsURL, "/api/metrics/stream")
	})

	mux.HandleFunc("/ws/replay_updates", func(w http.ResponseWriter, r *http.Request) {
		proxyWebSocket(w, r, cfg.ReplayURL, "/api/replay/stream")
	})

	mux.HandleFunc("/ws/notifications", func(w http.ResponseWriter, r *http.Request) {
		proxyWebSocket(w, r, cfg.ProjectURL, "/api/v1/projects/notifications/stream")
	})

	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		writeAPIError(w, http.StatusNotFound, "ROUTE_NOT_FOUND", "route not found")
	})

	// Initialize rate limiter: 600 requests/minute per key, burst of 100
	limiter := newRateLimiter(600, 100)
	// Periodic cleanup of stale rate limit buckets
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			limiter.cleanup()
		}
	}()

	// Build middleware chain: security headers → rate limit → CORS → JWT auth → size limit → logging → mux
	const maxBodySize int64 = 10 * 1024 * 1024 // 10MB max request body
	var handler http.Handler = mux
	handler = withLogging(handler, &requestsTotal)
	handler = withRequestSizeLimit(handler, maxBodySize)
	handler = withJWTAuth(handler, cfg.JWTSecret)
	handler = withCORS(handler, cfg.AllowedOrigin)
	handler = withRateLimit(handler, limiter)
	handler = withSecurityHeaders(handler)

	return &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      300 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
}

func withLogging(next http.Handler, requestsTotal *uint64) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddUint64(requestsTotal, 1)
		next.ServeHTTP(w, r)
		log.Printf("%s %s", r.Method, r.URL.Path)
	})
}

func hasBearerToken(raw string) bool {
	if raw == "" {
		return false
	}
	parts := strings.Split(raw, " ")
	if len(parts) != 2 {
		return false
	}
	if !strings.EqualFold(parts[0], "Bearer") {
		return false
	}
	return strings.TrimSpace(parts[1]) != ""
}

func withCORS(next http.Handler, allowedOrigin string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

type apiEnvelope struct {
	Success bool      `json:"success"`
	Data    any       `json:"data"`
	Error   *apiError `json:"error"`
}

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func writeAPIData(w http.ResponseWriter, status int, payload any) {
	writeEnvelope(w, status, apiEnvelope{Success: true, Data: payload, Error: nil})
}

func writeAPIError(w http.ResponseWriter, status int, code, message string) {
	writeEnvelope(w, status, apiEnvelope{Success: false, Data: nil, Error: &apiError{Code: code, Message: message}})
}

func writeEnvelope(w http.ResponseWriter, status int, envelope apiEnvelope) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(envelope)
}

// proxyWebSocket transparently proxies a WebSocket connection from the client
// to the upstream backend service. It copies frames bidirectionally.
func proxyWebSocket(w http.ResponseWriter, r *http.Request, backendURL, path string) {
	// Build the upstream WebSocket URL (ws:// scheme from http://).
	parsed, err := url.Parse(backendURL)
	if err != nil {
		http.Error(w, "bad upstream url", http.StatusBadGateway)
		return
	}
	scheme := "ws"
	if parsed.Scheme == "https" {
		scheme = "wss"
	}
	target := fmt.Sprintf("%s://%s%s", scheme, parsed.Host, path)
	if r.URL.RawQuery != "" {
		target += "?" + r.URL.RawQuery
	}

	// Dial the upstream backend WebSocket.
	backendConn, _, err := websocketDialer.Dial(target, nil)
	if err != nil {
		log.Printf("ws proxy dial %s: %v", target, err)
		http.Error(w, "upstream unavailable", http.StatusBadGateway)
		return
	}
	defer backendConn.Close()

	// Upgrade the client connection.
	clientConn, err := websocketUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer clientConn.Close()

	// Bidirectional copy.
	errc := make(chan error, 2)

	// Backend → Client
	go func() {
		for {
			msgType, msg, err := backendConn.ReadMessage()
			if err != nil {
				errc <- err
				return
			}
			if err := clientConn.WriteMessage(msgType, msg); err != nil {
				errc <- err
				return
			}
		}
	}()

	// Client → Backend
	go func() {
		for {
			msgType, msg, err := clientConn.ReadMessage()
			if err != nil {
				errc <- err
				return
			}
			if err := backendConn.WriteMessage(msgType, msg); err != nil {
				errc <- err
				return
			}
		}
	}()

	// Wait for either direction to close.
	<-errc
}
