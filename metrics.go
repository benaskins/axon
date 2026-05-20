package axon

import (
	"net/http"
	"strconv"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// RequestMetrics returns middleware that records HTTP request duration and
// count using OpenTelemetry instruments from the given MeterProvider.
//
// Instruments are created once when the constructor is called, so the
// same returned middleware reuses them across requests.
func RequestMetrics(mp metric.MeterProvider) func(http.Handler) http.Handler {
	meter := mp.Meter("axon")

	duration, err := meter.Float64Histogram(
		"http.server.request.duration",
		metric.WithDescription("HTTP request duration in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		panic("axon: failed to create duration histogram: " + err.Error())
	}

	total, err := meter.Int64Counter(
		"http.server.request.total",
		metric.WithDescription("Total HTTP requests by method, path, and status"),
	)
	if err != nil {
		panic("axon: failed to create request counter: " + err.Error())
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rw := wrapResponseWriter(w)
			next.ServeHTTP(rw, r)

			elapsed := time.Since(start).Seconds()
			path := r.Pattern
			if path == "" {
				path = r.URL.Path
			}
			status := strconv.Itoa(rw.statusCode)

			ctx := r.Context()
			duration.Record(ctx, elapsed,
				metric.WithAttributes(
					attribute.String("http.method", r.Method),
					attribute.String("http.route", path),
				),
			)
			total.Add(ctx, 1,
				metric.WithAttributes(
					attribute.String("http.method", r.Method),
					attribute.String("http.route", path),
					attribute.String("http.status_code", status),
				),
			)
		})
	}
}
