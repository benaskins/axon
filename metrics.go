package axon

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	promexporter "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"

	"go.opentelemetry.io/otel/attribute"
)

var (
	meterProvider *sdkmetric.MeterProvider

	httpRequestDuration metric.Float64Histogram
	httpRequestsTotal   metric.Int64Counter
)

func init() {
	exporter, err := promexporter.New()
	if err != nil {
		panic("axon: failed to create prometheus exporter: " + err.Error())
	}

	meterProvider = sdkmetric.NewMeterProvider(sdkmetric.WithReader(exporter))
	meter := meterProvider.Meter("axon")

	httpRequestDuration, err = meter.Float64Histogram(
		"http.server.request.duration",
		metric.WithDescription("HTTP request duration in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		panic("axon: failed to create duration histogram: " + err.Error())
	}

	httpRequestsTotal, err = meter.Int64Counter(
		"http.server.request.total",
		metric.WithDescription("Total HTTP requests by method, path, and status"),
	)
	if err != nil {
		panic("axon: failed to create request counter: " + err.Error())
	}
}

// MetricsHandler returns the Prometheus metrics HTTP handler.
// Serves OTel metrics in Prometheus exposition format.
func MetricsHandler() http.Handler {
	return promhttp.Handler()
}

// RequestMetrics returns middleware that records HTTP request metrics
// using OpenTelemetry instruments exported as Prometheus metrics.
func RequestMetrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := wrapResponseWriter(w)
		next.ServeHTTP(rw, r)

		duration := time.Since(start).Seconds()
		path := r.Pattern
		if path == "" {
			path = r.URL.Path
		}
		method := r.Method
		status := strconv.Itoa(rw.statusCode)

		ctx := r.Context()
		httpRequestDuration.Record(ctx, duration,
			metric.WithAttributes(
				attribute.String("http.method", method),
				attribute.String("http.route", path),
			),
		)
		httpRequestsTotal.Add(ctx, 1,
			metric.WithAttributes(
				attribute.String("http.method", method),
				attribute.String("http.route", path),
				attribute.String("http.status_code", status),
			),
		)
	})
}

// MeterProvider returns the global OTel MeterProvider, allowing domain
// packages to create their own meters and instruments.
func MeterProvider() *sdkmetric.MeterProvider {
	return meterProvider
}
