package main

import (
	"fmt"
	"github.com/hypebeast/go-osc/osc"
	"time"
)

func main() {
	ip := "localhost"
	port := 8765

	fmt.Println(fmt.Sprintf("Creating OSC Client - IP: %s, Port: %d", ip, port))

	// Create a new OSC client
	client := osc.NewOscClient(ip, port)
	// Create OSC message
	msg := osc.NewOscMessage("/test/address")
	// Append a new int32 argument
	msg.Append(int32(111))
	// Append True
	msg.Append(true)
	// Append a string value
	msg.Append("hello")
	// Send the message
	client.Send(msg)

	// Create an OSC bundle and add two OSC messages to it
	bundle := osc.NewOscBundle(time.Now())

	msg2 := osc.NewOscMessage("/test/bundle1")
	msg2.Append(int32(222))
	msg2.Append(true)
	bundle.Append(msg2)

	msg2 = osc.NewOscMessage("/test/bundle2")
	msg2.Append(int32(333))
	msg2.Append(false)
	bundle.Append(msg2)

	client.Send(bundle)
}
