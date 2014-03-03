package main

import (
	"fmt"
	"github.com/hypebeast/go-osc/osc"
	"time"
)

// TODO: Revise the client!
func main() {
	ip := "localhost"
	port := 8765

	fmt.Println(fmt.Sprintf("Creating OSC Client - IP: %s, Port: %d", ip, port))

	fmt.Println("Sending an OSC Message with some arguments...")

	client := osc.NewOscClient(ip, port)
	// Create OSC message
	msg := osc.NewOscMessage("/test/address")
	// Append a new int32 argument
	msg.Append(int32(111))
	// Append True
	// msg.Append(true)
	// Append a string value
	// msg.Append("hello")
	// Send the message
	client.Send(msg)

	fmt.Println("Sending an OSC bundle with two embedded OSC messages...")

	// Create an OSC bundle and add two OSC messages to it
	bundle := osc.NewOscBundle(time.Now())
	msg = osc.NewOscMessage("/test/bundle1")
	msg.Append(int32(222))
	// msg2.Append(true)
	bundle.Append(msg)

	msg = osc.NewOscMessage("/test/bundle2")
	msg.Append(int32(333))
	// msg2.Append(false)
	bundle.Append(msg)

	client.Send(bundle)
}
