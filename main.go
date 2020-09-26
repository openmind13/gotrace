package main

import (
	"log"
	"os"

	"github.com/openmind13/gotrace/trace"
)

func main() {

	if err := run(); err != nil {
		log.Fatal(err)
	}

}

func run() error {
	targetAddr := os.Args[1]

	tracer, err := trace.NewTracer(targetAddr)
	if err != nil {
		return err
	}

	if err = tracer.Start(); err != nil {
		return err
	}

	return nil
}
