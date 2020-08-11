package main

import (
	"github.com/openmind13/gotrace/trace"
)

var (
	targetAddr = "192.168.0.10"
)

func main() {
	// fmt.Printf("Gotrace\n")

	// if len(os.Args) < 2 {
	// 	os.Exit(0)
	// }

	// addr := os.Args[1]
	// fmt.Printf("%s\n", addr)

	// sendPacketWithTTL(addr, 1)
	// go listen("localhost")

	tracer, err := trace.NewTracer(targetAddr)
	if err != nil {
		panic(err)
	}

	err = tracer.Start()
	if err != nil {
		panic(err)
	}
}
