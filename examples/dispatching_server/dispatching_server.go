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

func printUsage() {
	fmt.Printf("Usage: %s PROTOCOL PORT\n", os.Args[0])
}

func main() {
	rand.Seed(time.Now().Unix())

	numArgs := len(os.Args[1:])

	if numArgs != 2 {
		printUsage()
		os.Exit(1)
	}

	var protocol osc.NetworkProtocol
	switch strings.ToLower(os.Args[1]) {
	case "udp":
		protocol = osc.UDP
	case "tcp":
		protocol = osc.TCP
	default:
		fmt.Println("Invalid protocol: " + os.Args[1])
		printUsage()
		os.Exit(1)
	}

	port, err := strconv.ParseInt(os.Args[2], 10, 32)
	if err != nil {
		fmt.Println(err)
		printUsage()
		os.Exit(1)
	}

	addr := fmt.Sprintf("127.0.0.1:%d", port)

	d := osc.NewStandardDispatcher()

	if err := d.AddMsgHandler("/message/address", func(msg *osc.Message) {
		osc.PrintMessage(msg)
	}); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	server := osc.NewServer(addr, d, 0,
		osc.WithProtocol(protocol))

	fmt.Printf("Listening via %s on port %d...\n", protocol, port)

	if err := server.ListenAndServe(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
