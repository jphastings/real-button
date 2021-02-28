package buttons

import (
	"github.com/jacobsa/go-serial/serial"
	"io"
	"time"
)

const buttonPressBufferSize = 16

func GetPort() (io.ReadWriteCloser, error) {
	options := serial.OpenOptions{
		// TODO: Select one automatically
		PortName:        "/dev/cu.usbserial-14220",
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

func AwaitPress(reader io.Reader) <-chan int {
	presses := make(chan int, buttonPressBufferSize)
	go func() {
		idx := make([]byte, 1)
		for {
			if _, err := reader.Read(idx); err != nil {
				close(presses)
			}
			// TODO: Any validation needed on button count
			presses <- int(idx[0])
		}
	}()
	return presses
}
