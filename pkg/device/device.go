package device

import (
	"github.com/jacobsa/go-serial/serial"
	"io"
	"time"
)

func GetPort(device string) (io.ReadWriteCloser, error) {
	// TODO: Discover automatically
	if device == "" {
		device = "/dev/ttyUSB0"
	}

	options := serial.OpenOptions{
		PortName:        device,
		BaudRate:        9600,
		DataBits:        8,
		StopBits:        1,
		MinimumReadSize: 1,
		ParityMode:      serial.PARITY_NONE,
	}

	port, err := serial.Open(options)
	if err != nil {
		return nil, err
	}

	// TODO: Why doesn't the first colour work unless we have a wait here?
	// TODO: Add dynamic wait here?
	<-time.After(1500 * time.Millisecond)

	return port, nil
}
