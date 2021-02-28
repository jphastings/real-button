package modules

import (
	"github.com/jphastings/vm-power/pkg/led"
)

type Module interface {
	Configure(map[string]interface{}) (<-chan led.State, chan<- Press, error)
	Close()
}

type Press struct{}
