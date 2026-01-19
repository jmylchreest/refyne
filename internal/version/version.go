// Package version provides build-time version information for refyne.
//
// Variables in this package are set at build time using ldflags:
//
//	go build -ldflags "-X github.com/jmylchreest/refyne/internal/version.Version=1.0.0 ..."
//
// For library consumers, the module version is determined by the go.mod
// and git tags (e.g., v1.0.0). This package exposes CLI build metadata.
package version

import (
	"fmt"
	"runtime"
	"strings"
)

// Build-time variables set via ldflags
var (
	// Version is the semantic version (e.g., "1.0.0" or "1.0.0-dev.5+abc123")
	Version = "dev"

	// Commit is the git commit SHA
	Commit = "unknown"

	// Dirty indicates if the working tree had uncommitted changes
	Dirty = "false"

	// BuildDate is the UTC build timestamp in RFC3339 format
	BuildDate = "unknown"
)

// Info contains structured version information
type Info struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	Dirty     bool   `json:"dirty"`
	BuildDate string `json:"build_date"`
	GoVersion string `json:"go_version"`
	Platform  string `json:"platform"`
}

// Get returns the current version information
func Get() Info {
	return Info{
		Version:   Version,
		Commit:    Commit,
		Dirty:     Dirty == "true",
		BuildDate: BuildDate,
		GoVersion: runtime.Version(),
		Platform:  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}
}

// String returns a single-line version string
func String() string {
	v := Version
	if Dirty == "true" {
		v += "-dirty"
	}
	return v
}

// Full returns a multi-line version string with all details
func Full() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("refyne %s\n", String()))
	sb.WriteString(fmt.Sprintf("  Commit:     %s\n", Commit))
	if Dirty == "true" {
		sb.WriteString("  Dirty:      yes\n")
	}
	sb.WriteString(fmt.Sprintf("  Built:      %s\n", BuildDate))
	sb.WriteString(fmt.Sprintf("  Go version: %s\n", runtime.Version()))
	sb.WriteString(fmt.Sprintf("  OS/Arch:    %s/%s", runtime.GOOS, runtime.GOARCH))
	return sb.String()
}
