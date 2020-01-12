package main

import (
	"bufio"
	"fmt"
	"io"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/hypebeast/go-osc/osc"
)

func testString() string {
	return "test string " + strconv.Itoa(rand.Intn(1000))
}

func testBlob() []byte {
	return []byte(testString())
}

var argFunctions = []func(*osc.Message){
	func(msg *osc.Message) {
		msg.Append(int32(rand.Intn(100000)))
	},
	func(msg *osc.Message) {
		msg.Append(rand.Float32() * 100000)
	},
	func(msg *osc.Message) {
		msg.Append(testString())
	},
	func(msg *osc.Message) {
		msg.Append(testBlob())
	},
	func(msg *osc.Message) {
		msg.Append(*osc.NewTimetag(time.Now()))
	},
	func(msg *osc.Message) {
		msg.Append(true)
	},
	func(msg *osc.Message) {
		msg.Append(false)
	},
	func(msg *osc.Message) {
		msg.Append(nil)
	},
}

func randomMessage(address string) *osc.Message {
	message := osc.NewMessage(address)

	for i := 0; i < 1+rand.Intn(5); i++ {
		argFunctions[rand.Intn(len(argFunctions))](message)
	}

	return message
}

func randomBundle() *osc.Bundle {
	bundle := osc.NewBundle(time.Now())

	for i := 0; i < 1+rand.Intn(5); i++ {
		if rand.Float32() < 0.25 {
			bundle.Append(randomBundle())
		} else {
			bundle.Append(randomMessage("/bundle/message/" + strconv.Itoa(i+1)))
		}
	}

	return bundle
}

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

	ip := "localhost"
	client := osc.NewClient(ip, int(port))
	client.SetNetworkProtocol(protocol)

	fmt.Println("### Welcome to go-osc transmitter demo")
	fmt.Println("Please, select the OSC packet type you would like to send:")
	fmt.Println("\tm: OSCMessage")
	fmt.Println("\tb: OSCBundle")
	fmt.Println("\tPress \"q\" to exit")
	fmt.Printf("# ")

	reader := bufio.NewReader(os.Stdin)
	for {
		line, err := reader.ReadBytes('\n')
		if err == io.EOF {
			break
		}

		if err != nil {
			fmt.Println("Error: " + err.Error())
			os.Exit(1)
		}

		sline := strings.TrimRight(string(line), "\n")
		if sline == "m" {
			if err := client.Send(randomMessage("/message/address")); err != nil {
				fmt.Println(err)
			}
		} else if sline == "b" {
			if err := client.Send(randomBundle()); err != nil {
				fmt.Println(err)
			}
		} else if sline == "q" {
			fmt.Println("Exit!")
			os.Exit(0)
		}

		fmt.Printf("# ")
	}
}
