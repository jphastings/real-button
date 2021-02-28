package device

import (
	"fmt"
	"github.com/jacobsa/go-serial/serial"
	"io"
	"log"
	"path/filepath"
)

var buttonIsSingular = map[bool]string{
	false: "buttons",
	true:  "button",
}

func GetPort() (io.ReadWriteCloser, int, error) {
	paths, err := filepath.Glob(possibleDevicesGlob)
	if err != nil {
		return nil, 0, err
	}
	for _, path := range paths {
		port, buttons, err := openSerial(path)
		if err != nil {
			continue
		}
		log.Printf("Found device with %d %s at %s\n", buttons, buttonIsSingular[buttons == 1], path)
		return port, buttons, nil
	}
	return nil, 0, fmt.Errorf("no devices seem to be available (searched %s)", possibleDevicesGlob)
}

func openSerial(path string) (io.ReadWriteCloser, int, error) {
	options := serial.OpenOptions{
		PortName:        path,
		BaudRate:        9600,
		DataBits:        8,
		StopBits:        1,
		MinimumReadSize: 1,
		ParityMode:      serial.PARITY_NONE,
	}

	port, err := serial.Open(options)
	if err != nil {
		return nil, 0, err
	}

	if _, err := port.Write([]byte{31}); err != nil {
		return nil, 0, err
	}

	resp := make([]byte, 1)
	if _, err := port.Read(resp); err != nil {
		port.Close()
		return nil, 0, err
	}

	if resp[0]&0b11100000 != 0b11100000 {
		port.Close()
		return nil, 0, fmt.Errorf("not a button device")
	}

	return port, int(resp[0] & 0b00011111), nil
}
