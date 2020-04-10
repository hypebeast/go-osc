package main

import "github.com/showcontroller/go-osc/osc"

func main() {
	addr := "127.0.0.1:8765"

	d := osc.NewStandardDispatcher()
	d.AddMsgHandler("/message/address", func(msg *osc.Message) {
		osc.PrintMessage(msg)
	})
	server := &osc.Server{
		Addr:       addr,
		Dispatcher: d,
	}

	server.ListenAndServe()
}
