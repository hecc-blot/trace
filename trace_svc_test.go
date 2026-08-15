package trace

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"github.com/stretchr/testify/assert"
)

func TestTraceStart(t *testing.T) {
	svc := &traceSvc{tracer: otel.Tracer("test")}
	ctx, span := svc.Start(context.Background(), "test-span", "key", "value")
	defer span.End()

	assert.NotNil(t, ctx)
	assert.Equal(t, "test-span", span.Name())
}

func TestTraceFromContext(t *testing.T) {
	svc := &traceSvc{tracer: otel.Tracer("test")}
	ctx, _ := svc.Start(context.Background(), "test-span")

	span := svc.FromContext(ctx)
	assert.NotNil(t, span)
}

func TestTraceInject(t *testing.T) {
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
	))
	svc := &traceSvc{tracer: otel.Tracer("test")}
	ctx, span := svc.Start(context.Background(), "test-span")
	defer span.End()

	t.Run("支持的 carrier", func(t *testing.T) {
		carrier := map[string]string{}
		assert.NoError(t, svc.Inject(ctx, carrier))
	})

	t.Run("不支持的 carrier", func(t *testing.T) {
		assert.Error(t, svc.Inject(ctx, "not a carrier"))
	})
}

func TestTraceExtract(t *testing.T) {
	svc := &traceSvc{tracer: otel.Tracer("test")}

	t.Run("支持的 carrier", func(t *testing.T) {
		ctx, err := svc.Extract(map[string]string{})
		assert.NoError(t, err)
		assert.NotNil(t, ctx)
	})

	t.Run("不支持的 carrier", func(t *testing.T) {
		_, err := svc.Extract("not a carrier")
		assert.Error(t, err)
	})
}
