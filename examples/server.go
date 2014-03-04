package main

import (
	"fmt"
	"github.com/hypebeast/go-osc/osc"
)

// TODO: Revise the server
func main() {
	address := "127.0.0.1"
	port := 8765
	server := osc.NewOscServer(address, port)

	server.AddMsgHandler("/test/address", func(msg *osc.OscMessage) {
		fmt.Println("Received message from " + msg.Address)
	})

	server.AddMsgHandler("/pattern1/matching", func(msg *osc.OscMessage) {
		fmt.Printf("Received message from '%s' matched '%s'\n", msg.Address, "/pattern1/matching")
	})

	server.AddMsgHandler("/patternx/matching", func(msg *osc.OscMessage) {
		fmt.Printf("Received message from '%s' matched '%s'\n", msg.Address, "/patternx/matching")
	})

	server.AddMsgHandler("/pattern/matching2/cat", func(msg *osc.OscMessage) {
		fmt.Printf("Received message from '%s' matched '%s'\n", msg.Address, "/pattern/matching2/cat")
	})

	server.AddMsgHandler("/pattern/matching2/dog", func(msg *osc.OscMessage) {
		fmt.Printf("Received message from '%s' matched '%s'\n", msg.Address, "/pattern/matching2/dog")
	})

	fmt.Printf("Listening on %s:%d\n", address, port)
	err := server.ListenAndServe()
	if err != nil {
		fmt.Println("Error")
	}
}
