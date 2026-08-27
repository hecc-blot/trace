package service

import (
	"context"

	iCoreApi "github.com/hecc-blot/framework/contract/api"
	traceEnum "github.com/hecc-blot/core/enum/trace"
	contract "github.com/hecc-blot/trace/contract"

	"github.com/gin-gonic/gin"
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

func (h *HttpTraceMiddleware) Middleware() any {
	return func(c *gin.Context) {
		span := startSpan(c, h.TraceSvc, "http.request",
			"http.method", c.Request.Method,
			"http.url", c.Request.URL.Path,
			"net.peer.ip", c.ClientIP(),
		)
		defer span.End()

		c.Next()
		span.SetAttribute("http.status_code", c.Writer.Status())
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

func (s *SseTraceMiddleware) Middleware() any {
	return func(c *gin.Context) {
		span := startSpan(c, s.TraceSvc, "sse.connection", "sse.path", c.FullPath())
		defer span.End()

		c.Next()
		span.SetAttribute("sse.status_code", c.Writer.Status())
	}
}

// startSpan 提取上游 trace、创建 span、注入 X-Trace-Id/traceparent 响应头，
// 并将 trace 上下文写回 request.Context，返回 span 供调用方 defer End。
func startSpan(c *gin.Context, traceSvc contract.ITrace, name string, attrs ...interface{}) contract.Span {
	carrier := make(propagation.HeaderCarrier)
	if traceparent := c.GetHeader("traceparent"); traceparent != "" {
		carrier.Set("traceparent", traceparent)
	}

	ctx, _ := traceSvc.Extract(carrier)
	ctx, span := traceSvc.Start(ctx, name, attrs...)

	traceID := ""
	if spanCtx := otelTrace.SpanFromContext(ctx).SpanContext(); spanCtx.HasTraceID() {
		traceID = spanCtx.TraceID().String()
		spanID := spanCtx.SpanID().String()
		span.SetAttribute("trace.id", traceID)
		c.Header("X-Trace-Id", traceID)
		c.Header("traceparent", "00-"+traceID+"-"+spanID+"-01")
	}

	ctx = context.WithValue(ctx, traceEnum.TraceIdKey, traceID)
	c.Request = c.Request.WithContext(ctx)
	return span
}
