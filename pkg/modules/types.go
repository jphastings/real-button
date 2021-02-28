package modules

import (
	"github.com/jphastings/vm-power/pkg/led"
)

type Module interface {
	Configure(map[string]interface{}) (Configured, error)
	Close()
}

type Configured struct {
	LEDState    <-chan led.State
	ButtonPress chan<- Press
	Run         func() error
}

type Press struct{}
