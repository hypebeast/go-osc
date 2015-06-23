package main

import osc "github.com/kward/go-osc"

func main() {
	addr := "0.0.0.0:8000"
	server := &osc.Server{Addr: addr}

	server.Handle("/message/address", func(msg *osc.Message) {
		osc.PrintMessage(msg)
	})

	server.ListenAndServe()
}
