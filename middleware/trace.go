package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/chihqiang/infra-go/httpx"
	"github.com/chihqiang/infra-go/trace"
	"go.opentelemetry.io/otel/codes"
)

type traceRecorder struct {
	http.ResponseWriter
	status int
}

func (rw *traceRecorder) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

// Trace 为每个请求创建根 span，并从上游请求头提取链路上下文，
// 使同一次调用在服务间共享同一 trace id，便于跨服务问题定位。
func Trace() httpx.Middleware {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			ctx, _ := trace.ExtractHeader(r.Context(), r.Header)

			ctx, span := trace.StartSpan(ctx, "http "+strings.ToLower(r.Method)+" "+r.URL.Path,
				trace.WithAttributes(
					trace.AttrString("http.method", r.Method),
					trace.AttrString("http.route", r.URL.Path),
				),
			)
			defer span.End()

			rw := &traceRecorder{ResponseWriter: w, status: http.StatusOK}
			next(rw, r.WithContext(ctx))

			span.SetAttributes(trace.AttrInt("http.status_code", rw.status))
			if rw.status >= 500 {
				span.SetStatus(codes.Error, http.StatusText(rw.status))
				span.RecordError(fmt.Errorf("http status %d", rw.status))
			}
		}
	}
}
