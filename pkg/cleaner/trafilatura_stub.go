//go:build !trafilatura

package cleaner

import "errors"

// ErrTrafilaturaNotAvailable is returned when trafilatura is used but not compiled in.
var ErrTrafilaturaNotAvailable = errors.New("trafilatura not available: build with -tags trafilatura to enable")

// OutputFormat specifies Trafilatura output format.
type OutputFormat int

const (
	// OutputHTML outputs cleaned HTML (default, for chaining with MarkdownCleaner).
	OutputHTML OutputFormat = iota
	// OutputText outputs plain text directly.
	OutputText
)

// Toggle specifies include/exclude behavior.
type Toggle int

const (
	// Default uses the default behavior for the field.
	Default Toggle = iota
	// Include explicitly includes the content.
	Include
	// Exclude explicitly excludes the content.
	Exclude
)

// TrafilaturaConfig configures the Trafilatura cleaner.
type TrafilaturaConfig struct {
	// Output format: OutputHTML (default) or OutputText
	Output OutputFormat
	// Comments: Include or Exclude (default: Exclude)
	Comments Toggle
	// Tables: Include or Exclude (default: Include)
	Tables Toggle
	// Links: Include or Exclude (default: Include)
	Links Toggle
	// Images: Include or Exclude (default: Include)
	Images Toggle
	// Fallback to Readability/DomDistiller: Include or Exclude (default: Include)
	Fallback Toggle
}

// TrafilaturaCleaner is a stub that returns an error when trafilatura is not compiled in.
// Build with -tags trafilatura to enable the real implementation.
type TrafilaturaCleaner struct{}

// NewTrafilatura returns a stub cleaner when trafilatura is not compiled in.
// The cleaner will return ErrTrafilaturaNotAvailable when Clean is called.
func NewTrafilatura(_ *TrafilaturaConfig) *TrafilaturaCleaner {
	return &TrafilaturaCleaner{}
}

// Clean returns an error indicating trafilatura is not available.
func (c *TrafilaturaCleaner) Clean(_ string) (string, error) {
	return "", ErrTrafilaturaNotAvailable
}

// Name returns the cleaner type.
func (c *TrafilaturaCleaner) Name() string {
	return "trafilatura"
}

// IsAvailable returns false when trafilatura is not compiled in.
func (c *TrafilaturaCleaner) IsAvailable() bool {
	return false
}
