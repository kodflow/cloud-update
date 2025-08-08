// Package telemetry provides OpenTelemetry integration for distributed tracing and metrics.
package telemetry

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/kodflow/cloud-update/src/internal/infrastructure/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
)

// Telemetry manages OpenTelemetry tracing and metrics.
type Telemetry struct {
	tracer         trace.Tracer
	meter          metric.Meter
	tracerProvider *sdktrace.TracerProvider
	meterProvider  *sdkmetric.MeterProvider
	
	// Metrics
	requestCounter   metric.Int64Counter
	requestDuration  metric.Float64Histogram
	activeJobs       metric.Int64UpDownCounter
	jobDuration      metric.Float64Histogram
	errorCounter     metric.Int64Counter
	rateLimitCounter metric.Int64Counter
}

// Config holds telemetry configuration.
type Config struct {
	Enabled      bool   // Whether telemetry is enabled
	ServiceName  string // Service name for traces
	OTLPEndpoint string // OTLP collector endpoint
	Environment  string // Environment (dev, staging, prod)
	Version      string // Service version
}

// LoadConfig loads telemetry configuration from environment.
func LoadConfig() *Config {
	return &Config{
		Enabled:      os.Getenv("CLOUD_UPDATE_OTEL_ENABLED") == "true",
		ServiceName:  getEnvOrDefault("CLOUD_UPDATE_SERVICE_NAME", "cloud-update"),
		OTLPEndpoint: getEnvOrDefault("CLOUD_UPDATE_OTEL_ENDPOINT", "localhost:4318"),
		Environment:  getEnvOrDefault("CLOUD_UPDATE_ENVIRONMENT", "development"),
		Version:      getEnvOrDefault("CLOUD_UPDATE_VERSION", "unknown"),
	}
}

// Initialize sets up OpenTelemetry with the given configuration.
func Initialize(ctx context.Context, cfg *Config) (*Telemetry, error) {
	if !cfg.Enabled {
		logger.Info("OpenTelemetry disabled")
		return &Telemetry{}, nil
	}

	// Create resource
	res, err := newResource(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Initialize tracer provider
	tracerProvider, err := newTracerProvider(ctx, cfg, res)
	if err != nil {
		return nil, fmt.Errorf("failed to create tracer provider: %w", err)
	}

	// Initialize meter provider
	meterProvider, err := newMeterProvider(ctx, cfg, res)
	if err != nil {
		return nil, fmt.Errorf("failed to create meter provider: %w", err)
	}

	// Set global providers
	otel.SetTracerProvider(tracerProvider)
	otel.SetMeterProvider(meterProvider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	// Create telemetry instance
	t := &Telemetry{
		tracer:         otel.Tracer(cfg.ServiceName),
		meter:          otel.Meter(cfg.ServiceName),
		tracerProvider: tracerProvider,
		meterProvider:  meterProvider,
	}

	// Initialize metrics
	if err := t.initMetrics(); err != nil {
		return nil, fmt.Errorf("failed to initialize metrics: %w", err)
	}

	logger.WithField("endpoint", cfg.OTLPEndpoint).Info("OpenTelemetry initialized")
	return t, nil
}

// newResource creates a new OpenTelemetry resource.
func newResource(cfg *Config) (*resource.Resource, error) {
	return resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceVersion(cfg.Version),
			semconv.DeploymentEnvironment(cfg.Environment),
			attribute.String("service.namespace", "cloud-update"),
		),
	)
}

// newTracerProvider creates a new tracer provider with OTLP exporter.
func newTracerProvider(ctx context.Context, cfg *Config, res *resource.Resource) (*sdktrace.TracerProvider, error) {
	// Create OTLP trace exporter
	exporter, err := otlptrace.New(ctx,
		otlptracehttp.NewClient(
			otlptracehttp.WithEndpoint(cfg.OTLPEndpoint),
			otlptracehttp.WithInsecure(), // Use TLS in production
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	// Create tracer provider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()), // Adjust sampling in production
	)

	return tp, nil
}

// newMeterProvider creates a new meter provider with OTLP exporter.
func newMeterProvider(ctx context.Context, cfg *Config, res *resource.Resource) (*sdkmetric.MeterProvider, error) {
	// Create OTLP metric exporter
	exporter, err := otlpmetrichttp.New(ctx,
		otlpmetrichttp.WithEndpoint(cfg.OTLPEndpoint),
		otlpmetrichttp.WithInsecure(), // Use TLS in production
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric exporter: %w", err)
	}

	// Create meter provider
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exporter,
			sdkmetric.WithInterval(10*time.Second))),
		sdkmetric.WithResource(res),
	)

	return mp, nil
}

// initMetrics initializes all metrics.
func (t *Telemetry) initMetrics() error {
	var err error

	// HTTP request counter
	t.requestCounter, err = t.meter.Int64Counter(
		"http.server.requests_total",
		metric.WithDescription("Total number of HTTP requests"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return fmt.Errorf("failed to create request counter: %w", err)
	}

	// HTTP request duration
	t.requestDuration, err = t.meter.Float64Histogram(
		"http.server.request_duration_seconds",
		metric.WithDescription("HTTP request duration in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return fmt.Errorf("failed to create request duration: %w", err)
	}

	// Active jobs gauge
	t.activeJobs, err = t.meter.Int64UpDownCounter(
		"jobs.active",
		metric.WithDescription("Number of active jobs"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return fmt.Errorf("failed to create active jobs gauge: %w", err)
	}

	// Job duration
	t.jobDuration, err = t.meter.Float64Histogram(
		"jobs.duration_seconds",
		metric.WithDescription("Job execution duration in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return fmt.Errorf("failed to create job duration: %w", err)
	}

	// Error counter
	t.errorCounter, err = t.meter.Int64Counter(
		"errors_total",
		metric.WithDescription("Total number of errors"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return fmt.Errorf("failed to create error counter: %w", err)
	}

	// Rate limit counter
	t.rateLimitCounter, err = t.meter.Int64Counter(
		"rate_limit_exceeded_total",
		metric.WithDescription("Total number of rate limit exceeded events"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return fmt.Errorf("failed to create rate limit counter: %w", err)
	}

	return nil
}

// StartSpan starts a new span with the given name.
func (t *Telemetry) StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	if t.tracer == nil {
		return ctx, noopSpan{}
	}
	return t.tracer.Start(ctx, name, opts...)
}

// RecordHTTPRequest records HTTP request metrics.
func (t *Telemetry) RecordHTTPRequest(ctx context.Context, method, path string, statusCode int, duration time.Duration) {
	if t.requestCounter == nil || t.requestDuration == nil {
		return
	}

	attrs := []attribute.KeyValue{
		attribute.String("http.method", method),
		attribute.String("http.route", path),
		attribute.Int("http.status_code", statusCode),
	}

	t.requestCounter.Add(ctx, 1, metric.WithAttributes(attrs...))
	t.requestDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))
}

// RecordJobStart records the start of a job.
func (t *Telemetry) RecordJobStart(ctx context.Context, jobID string, action string) {
	if t.activeJobs == nil {
		return
	}

	attrs := []attribute.KeyValue{
		attribute.String("job.id", jobID),
		attribute.String("job.action", action),
	}

	t.activeJobs.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// RecordJobComplete records the completion of a job.
func (t *Telemetry) RecordJobComplete(ctx context.Context, jobID string, action string, duration time.Duration, success bool) {
	if t.activeJobs == nil || t.jobDuration == nil {
		return
	}

	attrs := []attribute.KeyValue{
		attribute.String("job.id", jobID),
		attribute.String("job.action", action),
		attribute.Bool("job.success", success),
	}

	t.activeJobs.Add(ctx, -1, metric.WithAttributes(attrs...))
	t.jobDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))

	if !success && t.errorCounter != nil {
		t.errorCounter.Add(ctx, 1, metric.WithAttributes(
			attribute.String("error.type", "job_failure"),
			attribute.String("job.action", action),
		))
	}
}

// RecordError records an error.
func (t *Telemetry) RecordError(ctx context.Context, errorType string, err error) {
	if t.errorCounter == nil {
		return
	}

	attrs := []attribute.KeyValue{
		attribute.String("error.type", errorType),
		attribute.String("error.message", err.Error()),
	}

	t.errorCounter.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// RecordRateLimitExceeded records a rate limit exceeded event.
func (t *Telemetry) RecordRateLimitExceeded(ctx context.Context, clientIP string) {
	if t.rateLimitCounter == nil {
		return
	}

	t.rateLimitCounter.Add(ctx, 1, metric.WithAttributes(
		attribute.String("client.ip", clientIP),
	))
}

// Shutdown gracefully shuts down telemetry providers.
func (t *Telemetry) Shutdown(ctx context.Context) error {
	var err error

	if t.tracerProvider != nil {
		if shutdownErr := t.tracerProvider.Shutdown(ctx); shutdownErr != nil {
			err = fmt.Errorf("failed to shutdown tracer provider: %w", shutdownErr)
		}
	}

	if t.meterProvider != nil {
		if shutdownErr := t.meterProvider.Shutdown(ctx); shutdownErr != nil {
			if err != nil {
				err = fmt.Errorf("%w; failed to shutdown meter provider: %w", err, shutdownErr)
			} else {
				err = fmt.Errorf("failed to shutdown meter provider: %w", shutdownErr)
			}
		}
	}

	return err
}

// noopSpan is a no-op span for when telemetry is disabled.
type noopSpan struct{}

func (noopSpan) End(...trace.SpanEndOption) {}
func (noopSpan) AddEvent(string, ...trace.EventOption) {}
func (noopSpan) IsRecording() bool { return false }
func (noopSpan) RecordError(error, ...trace.EventOption) {}
func (noopSpan) SpanContext() trace.SpanContext { return trace.SpanContext{} }
func (noopSpan) SetStatus(trace.StatusCode, string) {}
func (noopSpan) SetName(string) {}
func (noopSpan) SetAttributes(...attribute.KeyValue) {}
func (noopSpan) TracerProvider() trace.TracerProvider { return nil }

// getEnvOrDefault returns the environment variable value or a default.
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}