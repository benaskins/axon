package axon

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/justinas/alice"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

// HealthCheck is a named health check function.
type HealthCheck struct {
	Name  string
	Check func() error
}

// HandlerOptions configures the observability wired into AppHandler.
// The zero value is valid: Logger falls back to slog.Default, MeterProvider
// falls back to otel.GetMeterProvider (a noop provider unless the composition
// root has set a global), and no health checks are run.
type HandlerOptions struct {
	Logger        *slog.Logger
	MeterProvider metric.MeterProvider
	HealthChecks  []HealthCheck
}

// AppHandler wraps the user handler with meta-header extraction, request
// logging, request metrics, and a /health endpoint. It does not mount
// /metrics — services that need pull-based metrics should run a separate
// exporter; FaaS deployments push via OTLP.
//
// The /health route is reserved: even if the inner handler registers it,
// the auto-wired version wins.
func AppHandler(handler http.Handler, opts HandlerOptions) http.Handler {
	logger := opts.Logger
	if logger == nil {
		logger = slog.Default()
	}
	mp := opts.MeterProvider
	if mp == nil {
		mp = otel.GetMeterProvider()
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", healthHandler(opts.HealthChecks))

	middleware := alice.New(
		MetaHeaders,
		RequestLogging(logger),
		RequestMetrics(mp),
	)
	mux.Handle("/", middleware.Then(handler))

	return mux
}

func healthHandler(checks []HealthCheck) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if len(checks) == 0 {
			json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
			return
		}

		status := http.StatusOK
		result := map[string]any{"status": "ok"}
		checkResults := make(map[string]string, len(checks))

		for _, hc := range checks {
			if err := hc.Check(); err != nil {
				checkResults[hc.Name] = err.Error()
				status = http.StatusServiceUnavailable
				result["status"] = "unhealthy"
			} else {
				checkResults[hc.Name] = "ok"
			}
		}

		result["checks"] = checkResults
		w.WriteHeader(status)
		json.NewEncoder(w).Encode(result)
	}
}
