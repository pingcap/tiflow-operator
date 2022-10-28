package env

import (
	"fmt"
	"os"
	"path/filepath"
)

// ExpandPath expands a relative path to a workspace specific one.
func ExpandPath(path ...string) string {
	return filepath.Join(append([]string{projectDir()}, path...)...)
}

// PrependToPath prepends the supplied path to PATH
func PrependToPath(path ...string) {
	os.Setenv("PATH", fmt.Sprintf("%s:%s", filepath.Join(path...), os.Getenv("PATH")))
}

// projectDir returns the project root directory, determined by bazel. Panics if the directory can't be determined.
func projectDir() string {
	return os.Getenv("TEST_WORKSPACE")
}
