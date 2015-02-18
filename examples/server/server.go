package main

import (
	"bufio"
	"fmt"
	"os"

	"github.com/hypebeast/go-osc/osc"
)

// TODO: Revise the server
func main() {
	address := "127.0.0.1"
	port := 8765
	server := osc.NewServer(address, port)

	fmt.Println("### Welcome to go-osc receiver demo")
	fmt.Println("Press \"q\" to exit")

	go func() {
		fmt.Printf("Start listening on \"%s:%d\"\n", address, port)
		server.Listen()

		for {
			packet, err := server.ReceivePacket()
			if err != nil {
				fmt.Println("Server error: " + err.Error())
				server.Close()
				os.Exit(1)
			}

			if packet != nil {
				switch packet.(type) {
				default:
					fmt.Println("Unknow packet type!")

				case *osc.Message:
					fmt.Printf("-- OSC Message: ")
					osc.PrintMessage(packet.(*osc.Message))

				case *osc.Bundle:
					fmt.Println("-- OSC Bundle:")
					bundle := packet.(*osc.Bundle)
					for i, message := range bundle.Messages {
						fmt.Printf("  -- OSC Message #%d: ", i+1)
						osc.PrintMessage(message)
					}
				}
			}
		}
	}()

	reader := bufio.NewReader(os.Stdin)

	for {
		c, err := reader.ReadByte()
		if err != nil {
			server.Close()
			os.Exit(0)
		}

		if c == 'q' {
			server.Close()
			os.Exit(0)
		}
	}
}
