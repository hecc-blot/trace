package trace

import (
	"context"
	"errors"

	traceContract "github.com/hecc-blot/hecc-blot-trace/contract"
	traceConf "github.com/hecc-blot/hecc-blot-trace/config"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	traceSDK "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	otelTrace "go.opentelemetry.io/otel/trace"
)

type traceSvc struct {
	tracer otelTrace.Tracer
}

type spanWrapper struct {
	span otelTrace.Span
	name string
}

func (s *spanWrapper) End() {
	s.span.End()
}

func (s *spanWrapper) SetAttribute(key string, value interface{}) {
	var attr attribute.KeyValue
	switch v := value.(type) {
	case string:
		attr = attribute.String(key, v)
	case int:
		attr = attribute.Int(key, v)
	case int64:
		attr = attribute.Int64(key, v)
	case bool:
		attr = attribute.Bool(key, v)
	case float64:
		attr = attribute.Float64(key, v)
	default:
		attr = attribute.String(key, string(value.([]byte)))
	}
	s.span.SetAttributes(attr)
}

func (s *spanWrapper) RecordError(err error) {
	s.span.RecordError(err)
	s.span.SetStatus(codes.Error, err.Error())
}

func (s *spanWrapper) Name() string {
	return s.name
}

func (t *traceSvc) Start(ctx context.Context, name string, attrs ...interface{}) (context.Context, traceContract.Span) {
	var otelAttrs []attribute.KeyValue
	for i := 0; i < len(attrs); i += 2 {
		if i+1 < len(attrs) {
			key := attrs[i].(string)
			switch v := attrs[i+1].(type) {
			case string:
				otelAttrs = append(otelAttrs, attribute.String(key, v))
			case int:
				otelAttrs = append(otelAttrs, attribute.Int(key, v))
			case int64:
				otelAttrs = append(otelAttrs, attribute.Int64(key, v))
			case bool:
				otelAttrs = append(otelAttrs, attribute.Bool(key, v))
			case float64:
				otelAttrs = append(otelAttrs, attribute.Float64(key, v))
			}
		}
	}
	ctx, span := t.tracer.Start(ctx, name, otelTrace.WithAttributes(otelAttrs...))
	return ctx, &spanWrapper{span: span, name: name}
}

func (t *traceSvc) FromContext(ctx context.Context) traceContract.Span {
	span := otelTrace.SpanFromContext(ctx)
	return &spanWrapper{span: span, name: span.SpanContext().SpanID().String()}
}

func (t *traceSvc) Inject(ctx context.Context, carrier interface{}) error {
	if c, ok := carrier.(map[string]string); ok {
		otel.GetTextMapPropagator().Inject(ctx, propagation.MapCarrier(c))
		return nil
	}
	return errors.New("unsupported carrier type")
}

func (t *traceSvc) Extract(carrier interface{}) (context.Context, error) {
	switch c := carrier.(type) {
	case map[string]string:
		ctx := otel.GetTextMapPropagator().Extract(context.Background(), propagation.MapCarrier(c))
		return ctx, nil
	case propagation.HeaderCarrier:
		ctx := otel.GetTextMapPropagator().Extract(context.Background(), c)
		return ctx, nil
	}
	return nil, errors.New("unsupported carrier type")
}

func NewTraceSvc(config *traceConf.Config) (traceContract.ITrace, func(), error) {
	exp, err := otlptracehttp.New(context.Background(),
		otlptracehttp.WithEndpoint(config.Endpoint),
		otlptracehttp.WithInsecure())
	if err != nil {
		return nil, func() {}, err
	}

	res, err := resource.New(context.Background(),
		resource.WithAttributes(
			semconv.ServiceName(config.ServiceName),
		))
	if err != nil {
		return nil, func() {}, err
	}

	var sampler traceSDK.Sampler
	switch config.Sampler.Type {
	case "always":
		sampler = traceSDK.AlwaysSample()
	case "never":
		sampler = traceSDK.NeverSample()
	case "probability":
		sampler = traceSDK.TraceIDRatioBased(config.Sampler.Ratio)
	default:
		sampler = traceSDK.AlwaysSample()
	}

	traceProvider := traceSDK.NewTracerProvider(
		traceSDK.WithResource(res),
		traceSDK.WithSampler(sampler),
		traceSDK.WithBatcher(exp),
	)

	otel.SetTracerProvider(traceProvider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return &traceSvc{
			tracer: otel.Tracer(config.ServiceName),
		}, func() {
			traceProvider.Shutdown(context.Background())
		}, nil
}
