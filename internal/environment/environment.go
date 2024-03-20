package environment

import (
	"os"
	"strings"
)

const (
	Env = "environments"
	Key = "environment"
)

// GetEnvironments retrieves environment data from a comma-separated environment variable.
// It returns a slice containing the environments.
func GetEnvironments() []string {
	environments := os.Getenv(Env)
	return strings.Split(environments, ",")
}
