package pipeline

import (
	"errors"
	"fmt"
	"time"

	"gdf/internal/drive"
)

// Config holds the settings that differ between environments.
type Config struct {
	// ThumbnailerFunction is the name or ARN of the function that makes thumbnails.
	ThumbnailerFunction string
	// ReviewTimeout is how long an upload may wait for a reviewer. It must be
	// at least a second: the SDK reads zero as "wait without a timeout".
	ReviewTimeout time.Duration
	// ReviewPolicy decides which uploads need a reviewer at all.
	ReviewPolicy drive.ReviewPolicy
}

// ErrInvalidConfig reports settings the pipeline cannot run with.
var ErrInvalidConfig = errors.New("invalid pipeline configuration")

func (c Config) validate() error {
	if c.ThumbnailerFunction == "" {
		return fmt.Errorf("%w: no thumbnailer function", ErrInvalidConfig)
	}

	if c.ReviewTimeout < time.Second {
		return fmt.Errorf("%w: review timeout is %s, want at least 1s", ErrInvalidConfig, c.ReviewTimeout)
	}

	return nil
}
