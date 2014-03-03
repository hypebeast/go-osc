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

	fmt.Printf("Listening on %s:%d\n", address, port)
	err := server.ListenAndServe()
	if err != nil {
		fmt.Println("Error")
	}
}
