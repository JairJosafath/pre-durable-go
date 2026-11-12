package pipeline_test

import (
	"errors"
	"testing"
	"time"

	"gdf/internal/pipeline"
)

func TestConfigurationIsCheckedBeforeAnyUploadArrives(t *testing.T) {
	t.Parallel()

	valid := pipeline.Config{ThumbnailerFunction: thumbnailerName, ReviewTimeout: time.Hour}

	tests := map[string]func(*pipeline.Config){
		"no thumbnailer function": func(c *pipeline.Config) { c.ThumbnailerFunction = "" },
		// The SDK reads a zero timeout as "wait for ever".
		"no review timeout":       func(c *pipeline.Config) { c.ReviewTimeout = 0 },
		"negative review timeout": func(c *pipeline.Config) { c.ReviewTimeout = -time.Hour },
	}

	for name, breakIt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			cfg := valid
			breakIt(&cfg)

			if _, err := pipeline.New(nil, nil, nil, cfg); !errors.Is(err, pipeline.ErrInvalidConfig) {
				t.Errorf("err = %v, want ErrInvalidConfig", err)
			}
		})
	}

	if _, err := pipeline.New(nil, nil, nil, valid); err != nil {
		t.Errorf("a valid configuration was refused: %v", err)
	}
}
