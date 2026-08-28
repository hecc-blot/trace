package service

import (
	"context"

	traceEnum "github.com/hecc-blot/core/enum/trace"
	iCoreApi "github.com/hecc-blot/framework/contract/api"
	contract "github.com/hecc-blot/trace/contract"

	"go.opentelemetry.io/otel/propagation"
	otelTrace "go.opentelemetry.io/otel/trace"
)

// HttpTraceMiddleware 为每个 HTTP 请求创建 span，并注入 X-Trace-Id 响应头。
type HttpTraceMiddleware struct {
	TraceSvc contract.ITrace
}

// NewHttpMiddleware 创建 HTTP 链路追踪中间件，供 api 在组装阶段注册。
func NewHttpMiddleware(traceSvc contract.ITrace) iCoreApi.IMiddleware {
	return &HttpTraceMiddleware{TraceSvc: traceSvc}
}

func (h *HttpTraceMiddleware) Middleware() iCoreApi.MiddlewareFunc {
	return func(ctx iCoreApi.IContext) {
		span := startSpan(ctx, h.TraceSvc, "http.request",
			"http.method", ctx.Method(),
			"http.url", ctx.Path(),
			"net.peer.ip", ctx.ClientIP(),
		)
		defer span.End()

		ctx.Next()
		span.SetAttribute("http.status_code", ctx.Status())
	}
}

// SseTraceMiddleware 为每个 SSE 连接创建 span，并注入 X-Trace-Id 响应头。
type SseTraceMiddleware struct {
	TraceSvc contract.ITrace
}

// NewSseMiddleware 创建 SSE 链路追踪中间件，供 sse 在组装阶段注册。
func NewSseMiddleware(traceSvc contract.ITrace) iCoreApi.IMiddleware {
	return &SseTraceMiddleware{TraceSvc: traceSvc}
}

func (s *SseTraceMiddleware) Middleware() iCoreApi.MiddlewareFunc {
	return func(ctx iCoreApi.IContext) {
		span := startSpan(ctx, s.TraceSvc, "sse.connection", "sse.path", ctx.FullPath())
		defer span.End()

		ctx.Next()
		span.SetAttribute("sse.status_code", ctx.Status())
	}
}

// startSpan 提取上游 trace、创建 span、注入 X-Trace-Id/traceparent 响应头，
// 并将 trace 上下文写回 request.Context，返回 span 供调用方 defer End。
func startSpan(ctx iCoreApi.IContext, traceSvc contract.ITrace, name string, attrs ...interface{}) contract.Span {
	carrier := make(propagation.HeaderCarrier)
	if traceparent := ctx.GetHeader("traceparent"); traceparent != "" {
		carrier.Set("traceparent", traceparent)
	}

	c, _ := traceSvc.Extract(carrier)
	c, span := traceSvc.Start(c, name, attrs...)

	traceID := ""
	if spanCtx := otelTrace.SpanFromContext(c).SpanContext(); spanCtx.HasTraceID() {
		traceID = spanCtx.TraceID().String()
		spanID := spanCtx.SpanID().String()
		span.SetAttribute("trace.id", traceID)
		ctx.Header("X-Trace-Id", traceID)
		ctx.Header("traceparent", "00-"+traceID+"-"+spanID+"-01")
	}

	c = context.WithValue(c, traceEnum.TraceIdKey, traceID)
	ctx.SetRequestContext(c)
	return span
}
