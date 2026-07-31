package config

import (
	"path"
	"testing"
)

// Test that the function returns a non-empty slice of strings
func TestResourceDirs(t *testing.T) {
	dirs := ResourceDirs("test")
	if len(dirs) == 0 {
		t.Errorf("Expected non-empty slice of strings, got %v", dirs)
	}

	for _, dir := range dirs {
		if path.Base(dir) != "test" {
			t.Errorf("Expected last component of path to be 'test', got %s", path.Base(dir))
		}
	}
}
