package config

import (
	"path"
	"strings"

	"git.sr.ht/~rjarry/aerc/lib/xdg"
)

// Set at build time
var (
	shareDir   string
	libexecDir string
)

func buildDefaultDirs() []string {
	var defaultDirs []string

	prefixes := []string{
		xdg.ConfigPath(),
		"~/.local/libexec",
		xdg.DataPath(),
	}

	// Add XDG_CONFIG_HOME and XDG_DATA_HOME
	for _, v := range prefixes {
		if v != "" {
			defaultDirs = append(defaultDirs, xdg.ExpandHome(v, "aerc"))
		}
	}

	// Trim null chars inserted post-build by systems like Conda
	shareDir := strings.TrimRight(shareDir, "\x00")
	libexecDir := strings.TrimRight(libexecDir, "\x00")

	// Add custom buildtime dirs
	if libexecDir != "" && libexecDir != "/usr/local/libexec/aerc" {
		defaultDirs = append(defaultDirs, xdg.ExpandHome(libexecDir))
	}
	if shareDir != "" && shareDir != "/usr/local/share/aerc" {
		defaultDirs = append(defaultDirs, xdg.ExpandHome(shareDir))
	}

	// Add fixed fallback locations
	defaultDirs = append(defaultDirs, "/usr/local/libexec/aerc")
	defaultDirs = append(defaultDirs, "/usr/local/share/aerc")
	defaultDirs = append(defaultDirs, "/usr/libexec/aerc")
	defaultDirs = append(defaultDirs, "/usr/share/aerc")

	return defaultDirs
}

var SearchDirs = buildDefaultDirs()

// ResourceDirs returns a list of directories to search for specific resources
func ResourceDirs(resource string) []string {
	var dirs []string

	for _, dir := range SearchDirs {
		dirs = append(dirs, path.Join(dir, resource))
	}

	return dirs
}
