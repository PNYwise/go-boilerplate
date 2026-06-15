package middlewares

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/attribute"
	oteltrace "go.opentelemetry.io/otel/trace"

	"go-boilerplate/internal/utils/logs"
)

type responseBodyWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (r responseBodyWriter) Write(b []byte) (int, error) {
	if r.body.Len() < 4096 {
		available := 4096 - r.body.Len()
		if len(b) > available {
			r.body.Write(b[:available])
		} else {
			r.body.Write(b)
		}
	}
	return r.ResponseWriter.Write(b)
}

func (r responseBodyWriter) WriteString(s string) (int, error) {
	if r.body.Len() < 4096 {
		available := 4096 - r.body.Len()
		if len(s) > available {
			r.body.WriteString(s[:available])
		} else {
			r.body.WriteString(s)
		}
	}
	return r.ResponseWriter.WriteString(s)
}

// RequestLogger creates a middleware for standardized request logging.
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// 1. Retrieve OTEL context created by otelgin middleware
		ctx := c.Request.Context()
		span := oteltrace.SpanFromContext(ctx)
		var traceID string
		if span.SpanContext().IsValid() {
			traceID = span.SpanContext().TraceID().String()
		}

		// 2. Set Trace ID to response headers
		if traceID != "" {
			c.Writer.Header().Set("X-Trace-Id", traceID)
		}

		// 3. Inject Start Time into context for duration calculations
		ctx = context.WithValue(ctx, logs.StartTimeKey, start)
		c.Request = c.Request.WithContext(ctx)

		// 4. Gather Request Headers
		// var reqHeadersBuilder strings.Builder
		// for k, v := range c.Request.Header {
		// 	if len(v) > 0 {
		// 		reqHeadersBuilder.WriteString(k)
		// 		reqHeadersBuilder.WriteString("=")
		// 		reqHeadersBuilder.WriteString(v[0])
		// 		reqHeadersBuilder.WriteString(";")
		// 	}
		// }

		// 5. Read request body (for application/json, log up to 4KB)
		var bodyStr string
		contentType := c.GetHeader("Content-Type")
		if strings.Contains(contentType, "application/json") && c.Request.Body != nil {
			bodyBytes, err := io.ReadAll(c.Request.Body)
			if err == nil {
				if len(bodyBytes) > 4096 {
					bodyStr = string(bodyBytes[:4096]) + " [TRUNCATED]"
				} else {
					bodyStr = string(bodyBytes)
				}
				// Restore body so subsequent handlers can read it
				c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			}
		}

		// 6. Log request started
		startAttrs := []attribute.KeyValue{
			attribute.String("http.method", c.Request.Method),
			attribute.String("http.url", c.Request.URL.String()),
			attribute.String("http.client_ip", c.ClientIP()),
			attribute.String("http.user_agent", c.Request.UserAgent()),
		}
		// if reqHeadersBuilder.Len() > 0 {
		// 	startAttrs = append(startAttrs, attribute.String("http.request.headers", reqHeadersBuilder.String()))
		// }
		if bodyStr != "" {
			startAttrs = append(startAttrs, attribute.String("http.request.body", bodyStr))
		}
		logs.LogInfo(ctx, fmt.Sprintf("HTTP Request Started: %s %s", c.Request.Method, c.Request.URL.Path), startAttrs...)

		// 7. Setup response body interceptor
		w := &responseBodyWriter{body: &bytes.Buffer{}, ResponseWriter: c.Writer}
		c.Writer = w

		// Process request
		c.Next()

		// 8. Log after request is processed
		duration := time.Since(start)

		// Gather response headers
		// var resHeadersBuilder strings.Builder
		// for k, v := range c.Writer.Header() {
		// 	if len(v) > 0 {
		// 		resHeadersBuilder.WriteString(k)
		// 		resHeadersBuilder.WriteString("=")
		// 		resHeadersBuilder.WriteString(v[0])
		// 		resHeadersBuilder.WriteString(";")
		// 	}
		// }

		resBodyStr := ""
		resContentType := c.Writer.Header().Get("Content-Type")
		if strings.Contains(resContentType, "application/json") && w.body.Len() > 0 {
			if w.body.Len() >= 4096 {
				resBodyStr = w.body.String() + " [TRUNCATED]"
			} else {
				resBodyStr = w.body.String()
			}
		}

		endAttrs := []attribute.KeyValue{
			attribute.Int("http.status_code", c.Writer.Status()),
			attribute.String("http.duration", duration.String()),
		}

		// if resHeadersBuilder.Len() > 0 {
		// 	endAttrs = append(endAttrs, attribute.String("http.response.headers", resHeadersBuilder.String()))
		// }
		if resBodyStr != "" {
			endAttrs = append(endAttrs, attribute.String("http.response.body", resBodyStr))
		}

		// Log using existing utility
		logs.LogInfo(ctx, fmt.Sprintf("HTTP Request Completed: %d %s %s", c.Writer.Status(), c.Request.Method, c.Request.URL.Path), endAttrs...)
	}
}
