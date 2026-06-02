package appkit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/jmoiron/sqlx"
)

// HealthCheck reports whether a single dependency is healthy. It should return
// nil when the dependency is reachable/usable and an error otherwise. The
// context carries a deadline — checks must honor it so a slow dependency can't
// hang the /health endpoint.
type HealthCheck func(ctx context.Context) error

// ProbeServerConfig configures a lightweight, standard-library HTTP server
// intended for services that don't expose a full API — workers, schedulers,
// FTP jobs, etc. — but still need liveness/readiness endpoints for k8s,
// load balancers, or uptime monitoring.
//
// For services that need real routing/binding/middleware, keep using Echo;
// this is deliberately minimal.
type ProbeServerConfig struct {
	Port        int                    // listen port
	ServiceName string                 // reported in the banner and health payload
	LogHTTP     bool                   // when true, log a line per request
	Checks      map[string]HealthCheck // readiness checks, keyed by dependency name
}

// healthTimeout bounds how long the /health endpoint will wait on its checks.
const healthTimeout = 3 * time.Second

// NewProbeServer builds an *http.Server exposing three routes:
//
//	GET /        plain HTML banner naming the service
//	GET /up      liveness — always 200 "up" while the process is serving
//	GET /health  readiness — runs every configured Check; 503 if any fails,
//	             otherwise 200, with a per-dependency breakdown in the body
//
// The returned server is configured but NOT started; the caller owns its
// lifecycle (ListenAndServe / Shutdown) so it can plug into whatever graceful
// shutdown pattern the service already uses.
func NewProbeServer(cfg ProbeServerConfig, logger Logger) *http.Server {
	if cfg.ServiceName == "" {
		cfg.ServiceName = "appkit-service"
	}

	mux := http.NewServeMux()

	// Exact-match root ({$} prevents this from also catching every other path).
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "<h3>%s server</h3>", cfg.ServiceName)
	})

	mux.HandleFunc("GET /up", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, "up")
	})

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), healthTimeout)
		defer cancel()

		services := make(map[string]string, len(cfg.Checks))
		overall := "healthy"
		for name, check := range cfg.Checks {
			if err := check(ctx); err != nil {
				services[name] = "unhealthy"
				overall = "unhealthy"
				logger.Warn("health check failed", "service", name, "err", err)
			} else {
				services[name] = "healthy"
			}
		}

		payload := map[string]any{
			"status":            overall,
			"service":           cfg.ServiceName,
			"timestamp":         time.Now(),
			"required_services": services,
		}

		code := http.StatusOK
		if overall == "unhealthy" {
			code = http.StatusServiceUnavailable
		}
		writeJSON(w, code, payload)
	})

	// Always recover from panics; layer request logging on top when enabled.
	var handler http.Handler = mux
	handler = recoverMiddleware(logger)(handler)
	if cfg.LogHTTP {
		handler = loggingMiddleware(logger)(handler)
	}

	return &http.Server{
		Addr:        fmt.Sprintf(":%d", cfg.Port),
		Handler:     handler,
		IdleTimeout: time.Minute,
		ReadTimeout: 15 * time.Second, // guard against slow header senders
	}
}

// DBHealthCheck adapts a sqlx.DB into a HealthCheck by pinging it. A nil handle
// is treated as unhealthy so a misconfigured/failed connection surfaces on
// /health rather than silently passing.
func DBHealthCheck(db *sqlx.DB) HealthCheck {
	return func(ctx context.Context) error {
		if db == nil {
			return errors.New("database connection is nil")
		}
		return db.PingContext(ctx)
	}
}

// writeJSON encodes v as JSON with the given status code. Encoding errors are
// logged-by-omission here (there's no logger in scope) but can't be sent to the
// client once the header is written, which matches typical handler behavior.
func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

// statusRecorder captures the response status code for request logging.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

// loggingMiddleware logs one structured line per request after it completes.
func loggingMiddleware(logger Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rec, r)
			logger.Info("request",
				"method", r.Method,
				"uri", r.RequestURI,
				"host", r.Host,
				"status", rec.status,
			)
		})
	}
}

// recoverMiddleware turns a handler panic into a 500 instead of crashing the
// process, logging the recovered value.
func recoverMiddleware(logger Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					logger.Error("panic recovered", "err", rec, "uri", r.RequestURI)
					http.Error(w, "internal server error", http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
