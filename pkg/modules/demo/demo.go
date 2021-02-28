package demo

import (
	"github.com/jphastings/vm-power/pkg/led"
	"github.com/jphastings/vm-power/pkg/modules"
	"time"
)

const pkgName = "demo"

func init() {
	modules.AvailableModules[pkgName] = New
}

var _ modules.Module = (*demoModule)(nil)

type demoModule struct {
	button <-chan modules.Press
	leds   chan<- led.State

	colorIdx int
}

func New(_config map[string]interface{}) (modules.Module, error) {
	return &demoModule{}, nil
}

func (d demoModule) Configure(_config map[string]interface{}) (modules.Configured, error) {
	button := make(chan modules.Press)
	d.button = button
	leds := make(chan led.State)
	d.leds = leds

	return modules.Configured{
		LEDState:    leds,
		ButtonPress: button,
		Run:         d.onPress,
	}, nil
}

func (d demoModule) onPress() error {
	for {
		for range d.button {
			d.leds <- colors[d.colorIdx]
			d.colorIdx = (d.colorIdx + 1) % len(colors)
		}
	}
	return nil
}

var colors = []led.State{
	{R: true, Flash: time.Second},
	{G: true, Flash: 500 * time.Millisecond},
	{B: true, Flash: 250 * time.Millisecond, TTL: 2 * time.Second},
}

func (d demoModule) Close() {
	close(d.leds)
}
