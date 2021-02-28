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

var domainEventMap = map[libvirt.DomainEventType]led.State{
	libvirt.DomainEventStarted:     led.Green,
	libvirt.DomainEventResumed:     led.Green,
	libvirt.DomainEventSuspended:   ledVMPaused,
	libvirt.DomainEventPmsuspended: ledVMPaused,
	libvirt.DomainEventStopped:     ledVMOff,
	libvirt.DomainEventShutdown:    ledVMOff,
	libvirt.DomainEventCrashed:     ledVMCrashed,
}

var domainStateName = map[libvirt.DomainState]string{
	libvirt.DomainRunning:     "running",
	libvirt.DomainPaused:      "paused",
	libvirt.DomainPmsuspended: "(power managed) suspended",
	libvirt.DomainShutdown:    "shutdown",
	libvirt.DomainShutoff:     "shutoff",
	libvirt.DomainCrashed:     "crashed",
}
var domainEventNames = map[libvirt.DomainEventType]string{
	libvirt.DomainEventStarted:     "started",
	libvirt.DomainEventResumed:     "resumed",
	libvirt.DomainEventSuspended:   "suspended",
	libvirt.DomainEventPmsuspended: "(power managed) suspended",
	libvirt.DomainEventStopped:     "stopped",
	libvirt.DomainEventShutdown:    "shutdown",
	libvirt.DomainEventCrashed:     "crashed",
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

	errors chan error
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
		errors:        make(chan error),
	}, nil
}

func (m *virtModule) Configure(config map[string]interface{}) (modules.Configured, error) {
	domainName, ok := config["domain"].(string)
	if !ok {
		return modules.Configured{}, fmt.Errorf("missing 'domain' key in virt config")
	}

	domain, err := m.virt.DomainLookupByName(domainName)
	if err != nil {
		return modules.Configured{}, fmt.Errorf("the domain %s does not exist: %w", domainName, err)
	}
	m.domain = domain

	leds := make(chan led.State)
	m.leds = leds
	button := make(chan modules.Press)
	m.button = button

	return modules.Configured{
		LEDState:    leds,
		ButtonPress: button,
		Run:         m.Run,
	}, nil
}

func (m *virtModule) Run() error {
	go m.registerDomainEvents()
	go m.onPress()
	return <-m.errors
}

func (m *virtModule) resetState() error {
	state, _, err := m.virt.DomainGetState(m.domain, getStateFlags)
	if err != nil {
		return err
	}

	st := libvirt.DomainState(state)
	log.Printf("Domain '%s' announced it is %s", m.domain.Name, domainStateName[st])

	if state, ok := domainStateMap[st]; ok {
		m.leds <- state
	}
	return nil
}

func (m *virtModule) registerDomainEvents() {
	if err := m.resetState(); err != nil {
		m.errors <- err
		return
	}

	events, err := m.virt.LifecycleEvents(context.Background())
	if err != nil {
		m.errors <- err
		return
	}

	for event := range events {
		if event.Dom.UUID != m.domain.UUID {
			continue
		}

		evt := libvirt.DomainEventType(event.Event)
		log.Printf("Domain '%s' announced it was %s\n", m.domain.Name, domainEventNames[evt])

		if state, ok := domainEventMap[evt]; ok {
			m.leds <- state
		}
	}
	m.errors <- fmt.Errorf("libvirt closed lifecycle event stream")
}

const (
	getStateFlags = 0
	wakeUpFlags   = 0
)

func (m *virtModule) onPress() {
	for {
		select {
		case <-m.button:
			state, _, err := m.virt.DomainGetState(m.domain, getStateFlags)
			if err != nil {
				m.leds <- led.Error
				m.errors <- err
				continue
			}

			switch libvirt.DomainState(state) {
			case libvirt.DomainRunning:
				m.confirm("shutdown", func() error { return m.virt.DomainShutdown(m.domain) })
			case libvirt.DomainPaused:
				log.Printf("Domain '%s' is being resumed", m.domain.Name)
				m.reportOutcome(m.virt.DomainResume(m.domain))
			case libvirt.DomainShutdown, libvirt.DomainShutoff, libvirt.DomainCrashed:
				log.Printf("Domain '%s' is being started", m.domain.Name)
				m.leds <- led.Blue
				m.reportOutcome(m.virt.DomainCreate(m.domain))
			case libvirt.DomainPmsuspended:
				log.Printf("Domain '%s' is being (power manage) woken up", m.domain.Name)
				m.reportOutcome(m.virt.DomainPmWakeup(m.domain, wakeUpFlags))
			case libvirt.DomainBlocked:
				log.Printf("Domain '%s' is blocked", m.domain.Name)
				m.leds <- led.Error
			default:
				log.Printf("Domain '%s' is in an unknown state: %v", m.domain.Name, state)
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
		log.Printf("Domain '%s' is being %s", m.domain.Name, name)
		if !m.reportOutcome(run()) {
			return
		}
		m.confirmations[name] = false
		return
	}

	log.Printf("Domain '%s' will be %s if confirmed", m.domain.Name, name)
	m.leds <- led.State{R: true, Flash: 150 * time.Millisecond}
	m.confirmations[name] = true
	go m.checkNotConfirmed(name)
}

func (m *virtModule) checkNotConfirmed(name string) {
	<-time.After(confirmTimeout)
	if m.confirmations[name] {
		m.confirmations[name] = false
		_ = m.resetState()
	}
}

func (m *virtModule) Close() {
	close(m.leds)
	if err := m.virt.Disconnect(); err != nil {
		m.errors <- err
	}
	close(m.errors)
}
