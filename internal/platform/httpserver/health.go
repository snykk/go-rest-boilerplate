package httpserver

import (
	"context"
	"log/slog"
	"net/http"
	"time"
)

// Check is one readiness probe. Probe returns an error when the dependency it checks cannot serve.
type Check struct {
	Name  string
	Probe func(ctx context.Context) error
}

// Health answers 200 {"status":"ok"} for as long as the process serves HTTP.
func Health() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}

// Ready runs every check within timeout. When all pass it answers 200
// {"status":"ok","checks":{"<name>":"ok"}}; when any fails it answers 503 with the status and that check
// set to "unavailable". A check's error is logged at WARN and never written to the response.
func Ready(log *slog.Logger, timeout time.Duration, checks ...Check) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), timeout)
		defer cancel()

		status, code := "ok", http.StatusOK
		results := make(map[string]string, len(checks))
		for _, check := range checks {
			if err := check.Probe(ctx); err != nil {
				log.LogAttrs(r.Context(), slog.LevelWarn, "readiness check failed",
					slog.String("check", check.Name), slog.String("error", err.Error()))
				results[check.Name] = "unavailable"
				status, code = "unavailable", http.StatusServiceUnavailable
				continue
			}
			results[check.Name] = "ok"
		}

		WriteJSON(w, code, map[string]any{"status": status, "checks": results})
	}
}
