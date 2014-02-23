package main

import (
	"fmt"
	"github.com/hypebeast/go-osc/osc"
)

func main() {
	server := osc.NewOscServer("127.0.0.1", 8766)
	err := server.ListenAndServe()
	if err != nil {
		fmt.Println("Error")
	}
}
