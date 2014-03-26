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
	msg.Append(int32(111))
	msg.Append(true)
	msg.Append("hello")
	client.Send(msg)

	fmt.Println("Sending an OSC bundle with two embedded OSC messages...")

	// Create an OSC bundle and add two OSC messages to it
	bundle := osc.NewOscBundle(time.Now().Add(time.Duration(5) * time.Second))
	msg = osc.NewOscMessage("/test/bundle1")
	msg.Append(int32(222))
	msg.Append(true)
	bundle.Append(msg)

	msg = osc.NewOscMessage("/test/bundle2")
	msg.Append(int32(333))
	msg.Append(false)
	bundle.Append(msg)
	client.Send(bundle)

	bundle = osc.NewOscBundle(time.Now().Add(time.Duration(7) * time.Second))
	msg = osc.NewOscMessage("/test/bundle3")
	msg.Append(int32(222))
	msg.Append(true)
	bundle.Append(msg)
	client.Send(bundle)

	msg = osc.NewOscMessage("/pattern?/matching")
	msg.Append(true)
	client.Send(msg)

	msg = osc.NewOscMessage("/pattern/matching2/*")
	msg.Append(true)
	client.Send(msg)
}
