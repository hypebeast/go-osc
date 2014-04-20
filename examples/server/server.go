package main

import (
	"bufio"
	"fmt"
	"github.com/hypebeast/go-osc/osc"
	"os"
)

// TODO: Revise the server
func main() {
	address := "127.0.0.1"
	port := 8765
	server := osc.NewOscServer(address, port)

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

				case *osc.OscMessage:
					fmt.Printf("-- OSC Message: ")
					osc.PrintOscMessage(packet.(*osc.OscMessage))

				case *osc.OscBundle:
					fmt.Println("-- OSC Bundle:")
					bundle := packet.(*osc.OscBundle)
					for i, message := range bundle.Messages {
						fmt.Printf("  -- OSC Message #%d: ", i+1)
						osc.PrintOscMessage(message)
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
