package main

import (
	"github.com/jphastings/vm-power/pkg"
	"log"
)

func main() {
	app, err := pkg.Load("./config.yaml")
	if err != nil {
		log.Fatal(err)
	}

	if err := app.Run("/dev/cu.usbserial-14230"); err != nil {
		log.Fatal(err)
	}
}
