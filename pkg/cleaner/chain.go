package cleaner

import (
	"strings"
)

// ChainCleaner applies multiple cleaners in sequence.
// This allows composing cleaners for multi-stage processing.
type ChainCleaner struct {
	cleaners []Cleaner
}

// NewChain creates a new cleaner that applies multiple cleaners in sequence.
// Cleaners are applied in the order provided.
//
// Example:
//
//	chain := cleaner.NewChain(
//	    refyne.New(refyne.PresetAggressive()),
//	    cleaner.NewNoop(),
//	)
func NewChain(cleaners ...Cleaner) *ChainCleaner {
	return &ChainCleaner{
		cleaners: cleaners,
	}
}

// Clean applies all cleaners in sequence.
func (c *ChainCleaner) Clean(content string) (string, error) {
	var err error
	for _, cleaner := range c.cleaners {
		content, err = cleaner.Clean(content)
		if err != nil {
			return "", err
		}
	}
	return content, nil
}

// Name returns the names of all chained cleaners.
func (c *ChainCleaner) Name() string {
	names := make([]string, len(c.cleaners))
	for i, cleaner := range c.cleaners {
		names[i] = cleaner.Name()
	}
	return "chain(" + strings.Join(names, "->") + ")"
}
