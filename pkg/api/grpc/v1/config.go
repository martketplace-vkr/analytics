package v1

import (
	"time"

	"github.com/martketplace-vkr/pkg/build/components"
)

type Config struct {
	DontRun bool
	Address string `validate:"required"`
	Retry   *Retry
	components.ComponentConfig
}

type Retry struct {
	Jitter     float64
	BaseDelay  time.Duration
	MaxDelay   time.Duration
	Multiplier float64
}
