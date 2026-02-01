package observability

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	oteltrace "go.opentelemetry.io/otel/trace"
)

var (
	tracer oteltrace.Tracer
	meter  metric.Meter
)

// InitTracing initializes OpenTelemetry tracing and metrics
func InitTracing(serviceName string, logger *slog.Logger) (func(), error) {
	// Create resource
	res, err := resource.New(context.Background(),
		resource.WithAttributes(
			attribute.String("service.name", serviceName),
			attribute.String("service.version", "1.0.0"),
		),
	)
	if err != nil {
		return nil, err
	}

	// Create Prometheus exporter for metrics
	promExporter, err := prometheus.New()
	if err != nil {
		return nil, err
	}

	// Create meter provider
	meterProvider := metric.NewMeterProvider(
		metric.WithReader(promExporter),
		metric.WithResource(res),
	)
	otel.SetMeterProvider(meterProvider)

	// Create trace provider
	tp := trace.NewTracerProvider(
		trace.WithResource(res),
		trace.WithSampler(trace.AlwaysSample()),
	)
	otel.SetTracerProvider(tp)

	tracer = tp.Tracer(serviceName)
	meter = meterProvider.Meter(serviceName)

	cleanup := func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			logger.Error("failed to shutdown tracer", "error", err)
		}
		if err := meterProvider.Shutdown(context.Background()); err != nil {
			logger.Error("failed to shutdown meter", "error", err)
		}
	}

	return cleanup, nil
}

// StartSpan starts a new span
func StartSpan(ctx context.Context, name string, opts ...oteltrace.SpanStartOption) (context.Context, oteltrace.Span) {
	return tracer.Start(ctx, name, opts...)
}

// SpanFromContext returns the current span from context
func SpanFromContext(ctx context.Context) oteltrace.Span {
	return oteltrace.SpanFromContext(ctx)
}

// AddEvent adds an event to the current span
func AddEvent(ctx context.Context, name string, attrs ...attribute.KeyValue) {
	span := SpanFromContext(ctx)
	if span != nil {
		span.AddEvent(name, oteltrace.WithAttributes(attrs...))
	}
}

// SetAttributes sets attributes on the current span
func SetAttributes(ctx context.Context, attrs ...attribute.KeyValue) {
	span := SpanFromContext(ctx)
	if span != nil {
		span.SetAttributes(attrs...)
	}
}

// RecordError records an error on the current span
func RecordError(ctx context.Context, err error) {
	span := SpanFromContext(ctx)
	if span != nil {
		span.RecordError(err)
	}
}

// Tracer returns the global tracer
func Tracer() oteltrace.Tracer {
	return tracer
}
