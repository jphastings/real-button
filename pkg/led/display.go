package led

import (
	"io"
	"sync"
	"time"
)

type PhysicalLED struct {
	index      uint8
	writer     io.Writer
	writeMutex *sync.Mutex

	state    *State
	disable  bool
	flashing bool
	reassess <-chan time.Time
}

func New(idx uint8, writer io.Writer, writeMutex *sync.Mutex) *PhysicalLED {
	return &PhysicalLED{
		index:      idx,
		writer:     writer,
		writeMutex: writeMutex,
	}
}

func (l *PhysicalLED) DisplayChanges(changes <-chan State) {
	for {
		select {
		case state := <-changes:
			l.state = &state
			l.disable = false
			l.flashing = false
		case <-l.reassess:
		}
		l.update()
		l.updateDynamicState()
	}
}

func (l *PhysicalLED) updateDynamicState() {
	if l.state.Flash == 0 {
		if l.disable {
			l.disable = false
			l.reassess = nil
			return
		}

		l.flashing = false
		if l.state.TTL != 0 {
			l.disable = true
			l.reassess = time.After(l.state.TTL)
		}
		return
	}

	delay := l.state.Flash / 2
	if l.state.TTL != 0 {
		l.state.TTL -= delay
		if l.state.TTL <= 0 {
			l.state.Flash = 0
			l.flashing = false
			l.disable = false // So it'll be disabled by the end of the function
		}
	}
	l.disable = !l.disable
	l.reassess = time.After(delay)
}

func (l *PhysicalLED) update() error {
	var instruction []byte
	if l.disable {
		instruction = []byte{Off.Byte(l.index)}
	} else {
		instruction = []byte{l.state.Byte(l.index)}
	}

	l.writeMutex.Lock()
	_, err := l.writer.Write(instruction)
	l.writeMutex.Unlock()
	return err
}
