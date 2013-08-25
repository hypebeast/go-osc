package main

import (
	"fmt"
	"github.com/hypebeast/go-osc/osc"
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
	client.SendMessage(msg)
}
