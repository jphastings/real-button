package main

import (
	"fmt"
	"github.com/jphastings/vm-power/pkg"
	"log"
	"os"
)

func main() {
	app, err := pkg.Load("./config.yaml")
	if err != nil {
		log.Fatal(err)
	}

	if err := app.Run(); err != nil {
		fmt.Fprintf(os.Stderr, err.Error())
	}
}
