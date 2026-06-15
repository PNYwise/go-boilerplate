package middlewares

import (
	"fmt"
	"go-boilerplate/internal/utils/logs"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
)

func CustomRecoveryMiddleware() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, err any) {
		tracer := otel.Tracer("panic")

		ctx, span := tracer.Start(c.Request.Context(), "panic")

		var parsedErr error
		switch e := err.(type) {
		case error:
			parsedErr = e
		case string:
			// It's a string, wrap it in an error
			parsedErr = fmt.Errorf("%s", e)
		default:
			parsedErr = fmt.Errorf("panic with unknown type: %v", e)
		}
		logs.SpanFatal(ctx, span, parsedErr, parsedErr.Error())
		defer span.End()
	})
}
