package main

import (
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/hypebeast/go-osc/osc"
)

func indent(str string, indentLevel int) string {
	indentation := strings.Repeat("  ", indentLevel)

	result := ""

	for i, line := range strings.Split(str, "\n") {
		if i != 0 {
			result += "\n"
		}

		result += indentation + line
	}

	return result
}

func debug(packet osc.Packet, indentLevel int) string {
	switch packet.(type) {
	default:
		return "Unknown packet type!"

	case *osc.Message:
		return fmt.Sprintf("-- OSC Message: %s", packet.(*osc.Message))

	case *osc.Bundle:
		bundle := packet.(*osc.Bundle)

		result := fmt.Sprintf("-- OSC Bundle (%s):", bundle.Timetag.Time())

		for i, message := range bundle.Messages {
			result += "\n" + indent(
				fmt.Sprintf("-- OSC Message #%d: %s", i+1, message),
				indentLevel+1,
			)
		}

		for _, bundle := range bundle.Bundles {
			result += "\n" + indent(debug(bundle, 0), indentLevel+1)
		}

		return result
	}
}

// Debugger is a simple Dispatcher that prints all messages and bundles as they
// are received.
type Debugger struct{}

// Dispatch implements Dispatcher.Dispatch by printing the packet received.
func (Debugger) Dispatch(packet osc.Packet) {
	if packet != nil {
		fmt.Println(debug(packet, 0) + "\n")
	}
}

func printUsage() {
	fmt.Printf("Usage: %s PORT\n", os.Args[0])
}

func main() {
	rand.Seed(time.Now().Unix())

	numArgs := len(os.Args[1:])

	if numArgs != 2 {
		printUsage()
		os.Exit(1)
	}

	port, err := strconv.ParseInt(os.Args[1], 10, 32)
	if err != nil {
		fmt.Println(err)
		printUsage()
		os.Exit(1)
	}

	addr := fmt.Sprintf("127.0.0.1:%d", port)

	server := &osc.Server{Addr: addr, Dispatcher: Debugger{}}

	fmt.Println("### Welcome to go-osc receiver demo")
	fmt.Printf("Listening via UDP on port %d...\n", port)

	if err := server.ListenAndServe(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
