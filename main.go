package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/sdk/resource"
	oteltrace "go.opentelemetry.io/otel/sdk/trace"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

var (
	tracer       trace.Tracer
	otlpEndpoint string
)

func init() {
	otlpEndpoint = os.Getenv("OTLP_ENDPOINT")
	if otlpEndpoint == "" {
		log.Fatalln("You MUST set OTLP_ENDPOINT env variable!")
	}
}



// OTLP Exporter
func newOTLPExporter(ctx context.Context) (oteltrace.SpanExporter, error) {
	exp, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint("http://localhost:4318/v1/traces")))
    if err != nil {
        log.Fatal(err)
    }
	return exp, err
}

func newTraceProvider(exp sdktrace.SpanExporter) *sdktrace.TracerProvider {
	// Ensure default SDK resources and the required service name are set.
	r, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName("ExampleService"),
		),
	)

	if err != nil {
		panic(err)
	}

	return sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(r),
	)
}

func main() {

	ctx := context.Background()

	exp, err := newOTLPExporter(ctx)
	if err != nil {
		log.Fatalf("failed to initialize exporter: %v", err)
	}

	r := echo.New()
	r.GET("/users/:id", func(c echo.Context) error {
		id := c.Param("id")
		name := getUser(c.Request().Context(), id)
		return c.JSON(http.StatusOK, struct {
			ID   string
			Name string
		}{
			ID:   id,
			Name: name,
		})
	})
	_ = r.Start(":8080")

	// Create a new tracer provider with a batch span processor and the given exporter.
	tp := newTraceProvider(exp)

	// Handle shutdown properly so nothing leaks.
	defer func() { _ = tp.Shutdown(ctx) }()

	otel.SetTracerProvider(tp)

	// Finally, set the tracer that can be used for this package.
	tracer = tp.Tracer("example.io/package/name")
}

func getUser(ctx context.Context, id string) string {
	ctx, span := tracer.Start(ctx, "HTTP GET /user")

	defer span.End()
	if id == "123" {
		return "otelecho tester"
	}

	db(ctx)
	// Add additional delay to simulate HTTP request.
	time.Sleep(1 * time.Second)

	return "unknown"
}

func db(ctx context.Context) {
	_, span := tracer.Start(ctx, "SQL SELECT")
	defer span.End()

	// Simulate Database call to SELECT connected devices.
	time.Sleep(2 * time.Second)
}