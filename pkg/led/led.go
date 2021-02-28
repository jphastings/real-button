package led

import "time"

var (
	Off   = State{}
	Red   = State{R: true}
	Green = State{G: true}
	Blue  = State{B: true}

	Error      = State{R: true, Flash: 333 * time.Millisecond, TTL: 4 * time.Second}
	Performing = State{B: true, Flash: 500 * time.Millisecond}
)

type State struct {
	R     bool
	G     bool
	B     bool
	Flash time.Duration

	TTL time.Duration
}

func (l State) Byte(number uint8) byte {
	// Only the lower 5 bits are button numbers
	cmd := number & 0b00011111
	if l.R {
		cmd += 0b10000000
	}
	if l.G {
		cmd += 0b01000000
	}
	if l.B {
		cmd += 0b00100000
	}
	return cmd
}
