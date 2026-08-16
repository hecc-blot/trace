package trace

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	traceSDK "go.opentelemetry.io/otel/sdk/trace"
)

func setupTraceSvc() *traceSvc {
	provider := traceSDK.NewTracerProvider(traceSDK.WithSampler(traceSDK.AlwaysSample()))
	otel.SetTracerProvider(provider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
	))
	return &traceSvc{tracer: otel.Tracer("test")}
}

func TestHttpTraceMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := setupTraceSvc()

	engine := gin.New()
	httpMw := NewHttpMiddleware(svc).Middleware().(func(*gin.Context))
	engine.Use(gin.HandlerFunc(httpMw))
	engine.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "pong")
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	engine.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.NotEmpty(t, recorder.Header().Get("X-Trace-Id"), "应注入 X-Trace-Id 响应头")
	assert.Regexp(t, `^00-[0-9a-f]{32}-[0-9a-f]{16}-01$`,
		recorder.Header().Get("traceparent"), "traceparent 应为 W3C Trace Context 格式")
}

func TestSseTraceMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := setupTraceSvc()

	engine := gin.New()
	sseMw := NewSseMiddleware(svc).Middleware().(func(*gin.Context))
	engine.Use(gin.HandlerFunc(sseMw))
	engine.GET("/events/time", func(c *gin.Context) {
		c.String(http.StatusOK, "data: ok\n\n")
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/events/time", nil)
	engine.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.NotEmpty(t, recorder.Header().Get("X-Trace-Id"), "SSE 连接也应注入 X-Trace-Id 响应头")
}
