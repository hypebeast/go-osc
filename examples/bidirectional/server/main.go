package main

import "github.com/hypebeast/go-osc/osc"

func main() {
	var server *osc.Server
	addr := "127.0.0.1:8765"
	d := osc.NewStandardDispatcher()
	d.AddMsgHandler("/message/address", func(msg *osc.Message) {
		osc.PrintMessage(msg)
		server.SendTo(osc.NewMessage("/reply"), msg.SenderAddr())
	})

	server = &osc.Server{
		Addr:       addr,
		Dispatcher: d,
	}
	server.ListenAndServe()
}
