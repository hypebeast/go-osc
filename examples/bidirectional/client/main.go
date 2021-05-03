package main

import "github.com/hypebeast/go-osc/osc"

func main() {
	finished := make(chan struct{})

	client := osc.NewClient("localhost", 8765)
	defer client.Close()

	d := osc.NewStandardDispatcher()
	d.AddMsgHandler("/reply", func(msg *osc.Message) {
		osc.PrintMessage(msg)
		finished <- struct{}{}
	})
	client.SetDispatcher(d)
	go client.ListenAndServe()

	msg := osc.NewMessage("/message/address")
	client.Send(msg)

	<-finished
}
