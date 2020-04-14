package main

import (
	"github.com/eiannone/keyboard"
	"log"

	//"net"
	"os"
	"time"
)
import "fmt"

//import "github.com/Lobaro/slip"
import "github.com/showcontroller/go-osc/osc"

//type TCPServer struct {
//	Addr string
//	Dispatch osc.Dispatcher
//}
//
//func (ts *TCPServer) ListenServe() {
//	listener, err := net.Listen("tcp", ts.Addr)
//	checkError(err)
//
//	for {
//		conn, err := listener.Accept()
//		if err != nil {
//			fmt.Println(err)
//			continue
//		}
//		go ts.HandleClient(conn)
//	}
//}
//
//func (ts *TCPServer) HandleClient(conn net.Conn) {
//	defer conn.Close()
//
//	fmt.Println("new connection from", conn.RemoteAddr())
//
//
//	r := slip.NewReader(conn)
//	for {
//
//		packet, _, err := r.ReadPacket()
//		if err != nil {
//			fmt.Println("error reading packet. stopping:", err)
//			return
//		}
//		//checkError(err)
//
//		fmt.Println("new packet", string(packet))
//		//checkError(err)
//		p, err := osc.ParsePacket(string(packet))
//		checkError(err)
//		//fmt.Printf("%T\n", p)
//		ts.Dispatch.Dispatch(p)
//		//fmt.Println("after", p)
//	}
//}
//
//type TCPClient struct {
//	addr string
//	conn net.Conn
//	w    *slip.Writer
//}
//
//func (tc *TCPClient) Send(msg *osc.Message) {
//	b, err := msg.MarshalBinary()
//	checkError(err)
//	tc.w.WritePacket(b)
//}
//
//func NewTCPClient(addr string) TCPClient {
//	t := TCPClient{}
//	t.addr = addr
//	c, err := net.Dial("tcp", t.addr)
//	checkError(err)
//	t.conn = c
//	w := slip.NewWriter(t.conn)
//	t.w = w
//	return t
//}

func main() {

	sd := osc.NewStandardDispatcher()
	err := sd.AddMsgHandler("*", func(msg *osc.Message) {
		log.Println("received a message ", msg.String())
		//osc.PrintMessage(msg)
	})
	if err != nil {
		log.Println("error adding message handler", err)
	}

	tc := osc.NewTCPClient("10.0.0.221:8765", sd)
	tc.Connect()
	go tc.Listen()

	m1 := osc.NewMessage("/led/1/high")
	m2 := osc.NewMessage("/led/1/low")
	m3 := osc.NewMessage("/led/2/high")
	m4 := osc.NewMessage("/led/2/low")
	m5 := osc.NewMessage("/led/3/high")
	m6 := osc.NewMessage("/led/3/low")

	t := 500 * time.Millisecond

	//tc.ReconnectWait = 1*time.Second

	err = keyboard.Open()
	if err != nil {
		panic(err)
	}

	fmt.Println("Press ESC to quit")
	for {
		char, key, err := keyboard.GetKey()
		fmt.Printf("You pressed: %q\r\n", char)
		if err != nil {
			panic(err)
		} else if key == keyboard.KeyEsc {
			break
		} else if char == '1' {
			tc.Send(m1)
		} else if char == '4' {
			tc.Send(m2)
		} else if char == '2' {
			tc.Send(m3)
		} else if char == '5' {
			tc.Send(m4)
		} else if char == '3' {
			tc.Send(m5)
		} else if char == '6' {
			tc.Send(m6)
		}
	}

	keyboard.Close()

	for {
		tc.Send(m1)
		time.Sleep(t)
		tc.Send(m2)
		time.Sleep(t)
		tc.Send(m3)
		time.Sleep(t)
		tc.Send(m4)
		time.Sleep(t)
		tc.Send(m5)
		time.Sleep(t)
		tc.Send(m6)
		time.Sleep(t)

	}

	//conn, _ := net.Dial("tcp", "10.0.0.221:8765")

	//w := slip.NewWriter(conn)
	// connect to this socket
	//m := osc.NewMessage("/led/1/high", "1")
	//fmt.Println(m.String())
	//b, _ := m.MarshalBinary()
	//w.WritePacket(b)

	//r := slip.NewReader(tc.conn)
	//packet, isPrefix, _ := r.ReadPacket()
	//fmt.Println(isPrefix, string(packet))

	//m = osc.NewMessage("/go")
	//b, _ = m.MarshalBinary()
	//w.WritePacket(b)
	//packet, isPrefix, _ = r.ReadPacket()

	//sd := osc.NewStandardDispatcher()
	//sd.AddMsgHandler("/howdy", func(msg *osc.Message) {
	//	fmt.Println("partner")
	//	osc.PrintMessage(msg)
	//})
	//fmt.Println("before", string(packet))
	//p, err := osc.ParsePacket(string(packet))
	//fmt.Println(err)
	//fmt.Printf("%T\n", p)
	//sd.Dispatch(p)
	//fmt.Println("after", p)
	//
	//fmt.Println(isPrefix, string(packet))

	//for {
	// read in input from stdin
	//reader := bufio.NewReader(os.Stdin)
	//fmt.Print("Text to send: ")
	//text, _ := reader.ReadString('\n')
	// send to socket
	//fmt.Fprintf(conn, text+"\n")
	// listen for reply
	// packet == [1, 2, 3]
	// isPrefix == false
	// err == io.EOF

	//message, _ := bufio.NewReader(conn).ReadString('\n')
	//fmt.Print("Message from server: " + message)
	//}
}

func checkError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %s", err.Error())
		os.Exit(1)
	}
}
