package dbs

import (
	"crypto/rand"
	"fmt"
)

// generateConnName generates a unique connection name for tracing/debugging
func generateConnName(appName, connType string) string {
	b := make([]byte, 4)
	rand.Read(b)
	return fmt.Sprintf("%s_%s_%x", appName, connType, b)
}
