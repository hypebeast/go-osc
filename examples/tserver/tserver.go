package main

import (
	"fmt"
	"github.com/Lobaro/slip"
	"github.com/showcontroller/go-osc/osc"
	"net"
	"os"
)

func main() {

	service := ":58888"
	listener, err := net.Listen("tcp", service)
	checkError(err)

	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		go handleClient(conn)
	}
}

func handleClient(conn net.Conn) {
	defer conn.Close()

	r := slip.NewReader(conn)
	packet, isPrefix, err := r.ReadPacket()
	checkError(err)
	fmt.Println(isPrefix, string(packet))

	sd := osc.NewStandardDispatcher()
	err = sd.AddMsgHandler("/alwaysReply", func(msg *osc.Message) {
		fmt.Println("hi")
		osc.PrintMessage(msg)
		w := slip.NewWriter(conn)
		// connect to this socket
		m := osc.NewMessage("/howdy", "1")
		fmt.Println(m.String())
		b, _ := m.MarshalBinary()
		err = w.WritePacket(b)
		checkError(err)
	})
	checkError(err)
	fmt.Println("before", string(packet))
	p, err := osc.ParsePacket(string(packet))
	fmt.Println(err)
	fmt.Printf("%T\n", p)
	sd.Dispatch(p)
	fmt.Println("after", p)

	//var buf [512]byte
	//for {
	//	n, err := conn.Read(buf[0:])
	//	if err != nil {
	//		return
	//	}
	//	_, err2 := conn.Write(buf[0:n])
	//	if err2 != nil {
	//		return
	//	}
	//}
}

func checkError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %s", err.Error())
		os.Exit(1)
	}
}
