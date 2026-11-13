package hooks

import (
	"fmt"
	"time"

	"github.com/seamanticus/aerc/config"
)

type AercShutdown struct {
	Lifetime time.Duration
}

func (a *AercShutdown) Cmd() string {
	return config.Hooks().AercShutdown
}

func (a *AercShutdown) Env() []string {
	return []string{
		fmt.Sprintf("AERC_LIFETIME=%s", a.Lifetime.String()),
	}
}
