package main

import "github.com/hypebeast/go-osc/osc"

func main() {
	addr := "127.0.0.1:8000"
	server := &osc.Server{Addr: addr}

	server.Handle("/message/address", func(msg *osc.Message) {
		osc.PrintMessage(msg)
	})

	server.ListenAndServe()
}
