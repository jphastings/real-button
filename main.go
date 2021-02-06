package main

import (
	"encoding/binary"
	"log"
	"os"
	"os/exec"
	"reflect"
	"time"
)

const (
	typeSync = 0x00
	typeKeyPress = 0x01
	debouncePeriod = 50 * time.Millisecond
	awaitDevicePause = 5 * time.Second
)

var (
	keyPress []uint16
	trigger = []uint16{125, 88}
	lastTriggered = time.Now()
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Please give a command to run when the button is pressed.")
	}

	for { openDeviceForPushes(runCommand) }
}

func runCommand() {
	cmd := exec.Command(os.Args[1], os.Args[2:]...)
	if err := cmd.Run(); err != nil {
		// TODO: Log stderr/stdout
		log.Println(err)
	} else {
		log.Printf("Button pressed, %s called", cmd.Path)
	}
}

func openDeviceForPushes(onPush func()) {
	dev, err := os.Open("/dev/input/event5")
	if err != nil {
		log.Printf("Couldn't open button device, waiting %v: %v\n", awaitDevicePause, err)
		time.Sleep(awaitDevicePause)
		return
	}
	defer dev.Close()

	if err := listenForPushes(dev, onPush); err != nil {
		log.Printf("Failed listening for pushes: %v\n", err)
	}
}

func listenForPushes(dev *os.File, onPush func()) error {
	log.Println("Listening for pushes")
	b := make([]byte, 24)
	for {
		n, err := dev.Read(b)
		if err != nil {
			return err
		}
		if n != len(b) {
			log.Printf("Only read %d bytes (instead of %d)\n", n, len(b))
			continue
		}

		pushed, err := decodePush(b)
		if err != nil {
			return err
		}

		if pushed && deBouncePush() {
			onPush()
		}
	}
}

func deBouncePush() bool {
	now := time.Now()
	triggered := now.Sub(lastTriggered) > debouncePeriod
	lastTriggered = now
	return triggered
}

func decodePush(b []byte) (bool, error) {
	// Useful code from https://janczer.github.io/work-with-dev-input/
	typ := binary.LittleEndian.Uint16(b[16:18])
	code := binary.LittleEndian.Uint16(b[18:20])

	switch typ {
	case typeSync:
		if code == 0 {
			same := reflect.DeepEqual(keyPress, trigger)
			keyPress = nil
			return  same, nil
		}
	case typeKeyPress:
		keyPress = append(keyPress, code)
	}

	return false, nil
}