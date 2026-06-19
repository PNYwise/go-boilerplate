package conn_name

import (
	"fmt"

	"github.com/rs/xid"
)

// generateConnName generates a unique connection name for tracing/debugging
func Generate(appName, connType string) string {
	return fmt.Sprintf("%s_%s_%s", appName, connType, xid.New().String())
}
