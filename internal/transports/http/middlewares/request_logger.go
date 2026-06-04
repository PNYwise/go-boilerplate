package middlewares

import (
	"bytes"
	"context"
	"io"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/xid"
	"go.opentelemetry.io/otel/attribute"

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

		// 1. Generate or extract Trace ID
		traceID := c.GetHeader("X-Trace-Id")
		if traceID == "" {
			traceID = "TID-" + xid.New().String()
		}

		// Generate Span ID
		spanID := "SID-" + xid.New().String()

		// 2. Set Trace ID to response headers
		c.Writer.Header().Set("X-Trace-Id", traceID)

		// 3. Inject Trace ID, Span ID, and Start Time into deep Go context
		ctx := context.WithValue(c.Request.Context(), logs.TraceIDKey, traceID)
		ctx = context.WithValue(ctx, logs.SpanIDKey, spanID)
		ctx = context.WithValue(ctx, logs.StartTimeKey, start)
		c.Request = c.Request.WithContext(ctx)

		// 4. Gather Request Headers
		var reqHeadersBuilder strings.Builder
		for k, v := range c.Request.Header {
			if len(v) > 0 {
				reqHeadersBuilder.WriteString(k)
				reqHeadersBuilder.WriteString("=")
				reqHeadersBuilder.WriteString(v[0])
				reqHeadersBuilder.WriteString(";")
			}
		}

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
		if reqHeadersBuilder.Len() > 0 {
			startAttrs = append(startAttrs, attribute.String("http.request.headers", reqHeadersBuilder.String()))
		}
		if bodyStr != "" {
			startAttrs = append(startAttrs, attribute.String("http.request.body", bodyStr))
		}
		logs.LogInfo(ctx, "HTTP Request Started", startAttrs...)

		// 7. Setup response body interceptor
		w := &responseBodyWriter{body: &bytes.Buffer{}, ResponseWriter: c.Writer}
		c.Writer = w

		// Process request
		c.Next()

		// 8. Log after request is processed
		duration := time.Since(start)

		// Gather response headers
		var resHeadersBuilder strings.Builder
		for k, v := range c.Writer.Header() {
			if len(v) > 0 {
				resHeadersBuilder.WriteString(k)
				resHeadersBuilder.WriteString("=")
				resHeadersBuilder.WriteString(v[0])
				resHeadersBuilder.WriteString(";")
			}
		}

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

		if resHeadersBuilder.Len() > 0 {
			endAttrs = append(endAttrs, attribute.String("http.response.headers", resHeadersBuilder.String()))
		}
		if resBodyStr != "" {
			endAttrs = append(endAttrs, attribute.String("http.response.body", resBodyStr))
		}

		// Log using existing utility
		logs.LogInfo(ctx, "HTTP Request Completed", endAttrs...)
	}
}
