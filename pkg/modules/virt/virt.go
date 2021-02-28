package virt

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/jphastings/vm-power/pkg/led"
	"github.com/jphastings/vm-power/pkg/modules"

	"github.com/digitalocean/go-libvirt"
)

const (
	pkgName        = "virt"
	offStateTTL    = 4 * time.Second
	confirmTimeout = 5 * time.Second
)

var (
	ledVMPaused  = led.State{R: true, G: true, TTL: offStateTTL, Flash: time.Second}
	ledVMCrashed = led.State{R: true, TTL: offStateTTL, Flash: 200 * time.Millisecond}
	ledVMOff     = led.State{R: true, TTL: offStateTTL}
)
var domainStateMap = map[libvirt.DomainState]led.State{
	libvirt.DomainRunning:     led.Green,
	libvirt.DomainPaused:      ledVMPaused,
	libvirt.DomainPmsuspended: ledVMPaused,
	libvirt.DomainShutdown:    ledVMOff,
	libvirt.DomainShutoff:     ledVMOff,
	libvirt.DomainCrashed:     ledVMCrashed,
}

// https://github.com/digitalocean/go-libvirt/blob/aaced3ae0e81/const.gen.go#L1262
var domainEventMap = map[libvirt.DomainEventType]led.State{
	libvirt.DomainEventStarted:     led.Green,
	libvirt.DomainEventResumed:     led.Green,
	libvirt.DomainEventSuspended:   ledVMPaused,
	libvirt.DomainEventPmsuspended: ledVMPaused,
	libvirt.DomainEventStopped:     ledVMOff,
	libvirt.DomainEventShutdown:    ledVMOff,
	libvirt.DomainEventCrashed:     ledVMCrashed,
}

func init() {
	modules.AvailableModules[pkgName] = New
}

var _ modules.Module = (*virtModule)(nil)

type virtModule struct {
	virt          *libvirt.Libvirt
	domain        libvirt.Domain
	confirmations map[string]bool

	button <-chan modules.Press
	leds   chan<- led.State
}

func New(config map[string]interface{}) (modules.Module, error) {
	address, ok := config["address"].(string)
	if !ok {
		return nil, fmt.Errorf("incorrect address given: %v", config["address"])
	}

	c, err := net.DialTimeout("tcp", address, 2*time.Second)
	if err != nil {
		return nil, err
	}

	// Drop a byte before libvirt.New(c)
	// More details at https://github.com/digitalocean/go-libvirt/issues/89
	// Removed as this issue does not exist any more.
	// c.Read(make([]byte, 1))

	l := libvirt.New(c)
	if err := l.Connect(); err != nil {
		return nil, err
	}

	return &virtModule{
		virt:          l,
		confirmations: make(map[string]bool),
	}, nil
}

func (m *virtModule) Configure(config map[string]interface{}) (<-chan led.State, chan<- modules.Press, error) {
	domainName, ok := config["domain"].(string)
	if !ok {
		return nil, nil, fmt.Errorf("missing 'domain' key in virt config")
	}

	domain, err := m.virt.DomainLookupByName(domainName)
	if err != nil {
		return nil, nil, fmt.Errorf("the domain %s does not exist: %w", domainName, err)
	}
	m.domain = domain

	leds := make(chan led.State)
	m.leds = leds
	go m.registerDomainEvents()

	button := make(chan modules.Press)
	m.button = button
	go m.onPress()

	return leds, button, nil
}

func (m *virtModule) resetState() {
	state, _, err := m.virt.DomainGetState(m.domain, getStateFlags)
	if err != nil {
		panic(err)
	}

	log.Println("Boot state:", state)

	if state, ok := domainStateMap[libvirt.DomainState(state)]; ok {
		m.leds <- state
	}
}

func (m *virtModule) registerDomainEvents() {
	m.resetState()

	events, err := m.virt.LifecycleEvents(context.Background())
	if err != nil {
		panic(err)
	}

	for event := range events {
		if event.Dom != m.domain {
			continue
		}

		if state, ok := domainEventMap[libvirt.DomainEventType(event.Event)]; ok {
			m.leds <- state
		}
	}
}

const (
	getStateFlags = 0
	wakeUpFlags   = 0
	rebootFlags   = 0
)

func (m *virtModule) onPress() {
	for {
		select {
		case <-m.button:
			state, _, err := m.virt.DomainGetState(m.domain, getStateFlags)
			if err != nil {
				panic(err)
			}

			fmt.Println("Current State:", state)

			switch libvirt.DomainState(state) {
			case libvirt.DomainRunning:
				m.confirm("shutdown", func() error { return m.virt.DomainShutdown(m.domain) })
			case libvirt.DomainPaused:
				m.reportOutcome(m.virt.DomainResume(m.domain))
			case libvirt.DomainShutdown, libvirt.DomainShutoff, libvirt.DomainCrashed:
				m.leds <- led.Error
			case libvirt.DomainPmsuspended:
				m.reportOutcome(m.virt.DomainPmWakeup(m.domain, wakeUpFlags))
			case libvirt.DomainBlocked:
				m.leds <- led.Error
			}
		}
	}
}

// reportOutcome sends a "performing" or "error" signal to the LEDs, based on the presence of the error.
// returns true if there was an error.
func (m *virtModule) reportOutcome(err error) (ok bool) {
	if err == nil {
		m.leds <- led.Performing
		return true
	}

	m.leds <- led.Error
	log.Println(err)
	return false
}

func (m *virtModule) confirm(name string, run func() error) {
	if confirmed, present := m.confirmations[name]; present && confirmed {
		if !m.reportOutcome(run()) {
			return
		}
		m.confirmations[name] = false
		return
	}

	m.leds <- led.State{R: true, Flash: 150 * time.Millisecond}
	m.confirmations[name] = true
	go m.checkNotConfirmed(name)
}

func (m *virtModule) checkNotConfirmed(name string) {
	<-time.After(confirmTimeout)
	if m.confirmations[name] {
		m.confirmations[name] = false
		m.resetState()
	}
}

func (m *virtModule) Close() {
	close(m.leds)
	_ = m.virt.Disconnect()
}
