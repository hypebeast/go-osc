package osc

import (
	"github.com/Lobaro/slip"
	"log"
	"net"
	//"os"
	"time"
)

type TCPServer struct {
	Addr     string
	Dispatch Dispatcher
}

// ListenServe listens on the TCP connection and stats a goroutine to handle each client
func (ts *TCPServer) ListenServe() error {
	listener, err := net.Listen("tcp", ts.Addr)
	//checkError(err)
	if err != nil {
		return err
	}

	// close the listener down at the end
	defer func() {
		err = listener.Close()
		if err != nil {
			log.Println(err)
		}
	}()

	// listen and handle new connections
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println(err)
			return err
			//continue
		}
		go ts.HandleClient(conn)
	}
	//return nil
}

// HandleClient takes an incoming connection and dispatches messages to the handles
func (ts *TCPServer) HandleClient(conn net.Conn) {
	defer func() {
		log.Println("closing connection to client", conn.RemoteAddr())
		err := conn.Close()
		if err != nil {
			log.Println("error closing connection", err)
		}
	}()

	log.Println("new connection from", conn.RemoteAddr())

	var r = slip.NewReader(conn)
	for {

		packet, _, err := r.ReadPacket()
		if err != nil {
			log.Println("error reading packet. stopping:", conn.RemoteAddr(), err)
			return
		}
		//checkError(err)

		log.Println("new packet", string(packet))
		//checkError(err)
		p, err := ParsePacket(string(packet))
		//checkError(err)
		if err != nil {
			log.Println("error parsing packet", string(packet))
		}
		//fmt.Printf("%T\n", p)
		ts.Dispatch.Dispatch(p)
		//fmt.Println("after", p)
	}
}

type TCPClient struct {
	addr          string
	conn          net.Conn
	w             *slip.Writer
	r             *slip.Reader // Add read function to tcpclient
	dispatcher    Dispatcher
	ReconnectWait time.Duration
}

// Send sends an OSC pkt (a message or bundle) over the TCP connection of the client
func (tc *TCPClient) Send(pkt Packet) {
	b, err := pkt.MarshalBinary()
	if err != nil {
		log.Println("error marshalling binary of packet")
	}
	//checkError(err)
	err = tc.w.WritePacket(b)
	if err != nil {
		log.Println("error writing packet", err)
		log.Println("attempting to reconnect...")
		tc.Connect()
	}
	//checkError(err)
}

// C
func (tc *TCPClient) Connect() {
	log.Println("connecting to", tc.addr)
	for {
		c, err := net.Dial("tcp", tc.addr)
		if err != nil {
			log.Println("error connecting.", err)
			log.Println("trying again after", tc.ReconnectWait)
			time.Sleep(tc.ReconnectWait)
			continue
		}
		//checkError(err)
		tc.conn = c
		log.Println("connected to", c.RemoteAddr())
		w := slip.NewWriter(tc.conn)
		tc.w = w
		r := slip.NewReader(tc.conn)
		tc.r = r
		//go tc.Listen()
		break
	}
}

func (tc *TCPClient) Listen() {
	for {
		log.Println("trying to read a packet")
		packet, _, err := tc.r.ReadPacket()
		if err != nil {
			log.Println("error reading packet. stopping for 200ms:", tc.conn.RemoteAddr(), err)
			time.Sleep(time.Millisecond * 200)
			continue
		}
		//checkError(err)

		log.Println("new packet", string(packet))
		//checkError(err)
		p, err := ParsePacket(string(packet))
		//checkError(err)
		if err != nil {
			log.Println("error parsing packet", string(packet))
		}
		//fmt.Printf("%T\n", p)
		tc.dispatcher.Dispatch(p)
		//fmt.Println("after", p)
	}
}

func NewTCPClient(addr string, d Dispatcher) TCPClient {
	t := TCPClient{}
	t.ReconnectWait = 200 * time.Millisecond
	t.addr = addr
	t.dispatcher = d

	return t
}
