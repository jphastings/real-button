package buttons

import "io"

const buttonPressBufferSize = 16

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
