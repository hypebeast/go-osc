package osc

import (
	"fmt"
	"github.com/Lobaro/slip"
	"net"
	"os"
	"time"
)

type TCPServer struct {
	Addr     string
	Dispatch Dispatcher
}

func (ts *TCPServer) ListenServe() error {
	listener, err := net.Listen("tcp", ts.Addr)
	checkError(err)
	if err != nil {
		return err
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println(err)
			return err
			//continue
		}
		go ts.HandleClient(conn)
	}
	return nil
}

func (ts *TCPServer) HandleClient(conn net.Conn) {
	defer func() {
		fmt.Println("closing connection to client", conn.RemoteAddr())
		err := conn.Close()
		if err != nil {
			fmt.Println("error closing connection", err)
		}
	}()

	fmt.Println("new connection from", conn.RemoteAddr())

	r := slip.NewReader(conn)
	for {

		packet, _, err := r.ReadPacket()
		if err != nil {
			fmt.Println("error reading packet. stopping:", conn.RemoteAddr(), err)
			return
		}
		//checkError(err)

		fmt.Println("new packet", string(packet))
		//checkError(err)
		p, err := ParsePacket(string(packet))
		checkError(err)
		//fmt.Printf("%T\n", p)
		ts.Dispatch.Dispatch(p)
		//fmt.Println("after", p)
	}
}

type TCPClient struct {
	addr          string
	conn          net.Conn
	w             *slip.Writer
	ReconnectWait time.Duration
}

func (tc *TCPClient) Send(msg *Message) {
	b, err := msg.MarshalBinary()
	checkError(err)
	err = tc.w.WritePacket(b)
	if err != nil {
		fmt.Println(err)
		tc.Reconnect()
	}
	//checkError(err)
}

func (tc *TCPClient) Reconnect() {
	fmt.Println("reconnecting to", tc.addr)
	for {
		c, err := net.Dial("tcp", tc.addr)
		if err != nil {
			fmt.Println("error connecting.", err)
			fmt.Println("trying again after", tc.ReconnectWait)
			time.Sleep(tc.ReconnectWait)
			continue
		}
		//checkError(err)
		tc.conn = c
		fmt.Println("connected to", c.RemoteAddr())
		w := slip.NewWriter(tc.conn)
		tc.w = w
		break
	}
}

func NewTCPClient(addr string) TCPClient {
	t := TCPClient{}
	t.ReconnectWait = 200 * time.Millisecond
	t.addr = addr
	c, err := net.Dial("tcp", t.addr)
	checkError(err)
	fmt.Println("connected to", c.RemoteAddr())
	t.conn = c
	w := slip.NewWriter(t.conn)
	t.w = w
	return t
}

func checkError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %s", err.Error())
		os.Exit(1)
	}
}
