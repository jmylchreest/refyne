package fetcher

import (
	"os/exec"

	"github.com/refyne/refyne/internal/logger"
)

// Common Chrome/Chromium binary names across different systems
var chromeBinaryNames = []string{
	"google-chrome-stable",
	"google-chrome",
	"chromium",
	"chromium-browser",
	"chrome",
	// macOS paths
	"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
	"/Applications/Chromium.app/Contents/MacOS/Chromium",
	// Common Linux paths
	"/usr/bin/google-chrome-stable",
	"/usr/bin/google-chrome",
	"/usr/bin/chromium",
	"/usr/bin/chromium-browser",
	"/snap/bin/chromium",
	// Windows paths
	`C:\Program Files\Google\Chrome\Application\chrome.exe`,
	`C:\Program Files (x86)\Google\Chrome\Application\chrome.exe`,
}

// FindChromePath searches for a Chrome/Chromium binary on the system.
// It first tries PATH lookup, then checks common installation locations.
// Returns empty string if no Chrome binary is found.
func FindChromePath() string {
	for _, name := range chromeBinaryNames {
		// Try PATH lookup first (for short names)
		if path, err := exec.LookPath(name); err == nil {
			logger.Debug("found Chrome binary", "name", name, "path", path)
			return path
		}
		// For absolute paths, check if executable exists
		if len(name) > 0 && (name[0] == '/' || (len(name) > 1 && name[1] == ':')) {
			if path, err := exec.LookPath(name); err == nil {
				logger.Debug("found Chrome binary", "path", path)
				return path
			}
		}
	}
	logger.Warn("no Chrome binary found - dynamic fetch mode may not work")
	return ""
}
