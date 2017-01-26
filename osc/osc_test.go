package osc

import (
	"bufio"
	"bytes"
	"net"
	"reflect"
	"sync"
	"testing"
	"time"
)

func TestMessage_Append(t *testing.T) {
	oscAddress := "/address"
	message := NewMessage(oscAddress)
	if message.Address != oscAddress {
		t.Errorf("OSC address should be \"%s\" and is \"%s\"", oscAddress, message.Address)
	}

	message.Append("string argument")
	message.Append(123456789)
	message.Append(true)

	if message.CountArguments() != 3 {
		t.Errorf("Number of arguments should be %d and is %d", 3, message.CountArguments())
	}
}

func TestMessage_Equals(t *testing.T) {
	msg1 := NewMessage("/address")
	msg2 := NewMessage("/address")
	msg1.Append(1234)
	msg2.Append(1234)
	msg1.Append("test string")
	msg2.Append("test string")

	if !msg1.Equals(msg2) {
		t.Error("Messages should be equal")
	}
}

func TestMessage_TypeTags(t *testing.T) {
	for _, tt := range []struct {
		desc string
		msg  *Message
		tags string
		ok   bool
	}{
		{"addr_only", NewMessage("/"), ",", true},
		{"nil", NewMessage("/", nil), ",N", true},
		{"bool_true", NewMessage("/", true), ",T", true},
		{"bool_false", NewMessage("/", false), ",F", true},
		{"int32", NewMessage("/", int32(1)), ",i", true},
		{"int64", NewMessage("/", int64(2)), ",h", true},
		{"float32", NewMessage("/", float32(3.0)), ",f", true},
		{"float64", NewMessage("/", float64(4.0)), ",d", true},
		{"string", NewMessage("/", "5"), ",s", true},
		{"[]byte", NewMessage("/", []byte{'6'}), ",b", true},
		{"two_args", NewMessage("/", "123", int32(456)), ",si", true},
		{"invalid_msg", nil, "", false},
		{"invalid_arg", NewMessage("/foo/bar", 789), "", false},
	} {
		tags, err := tt.msg.TypeTags()
		if err != nil && tt.ok {
			t.Errorf("%s: TypeTags() unexpected error: %s", tt.desc, err)
			continue
		}
		if err == nil && !tt.ok {
			t.Errorf("%s: TypeTags() expected an error")
			continue
		}
		if !tt.ok {
			continue
		}
		if got, want := tags, tt.tags; got != want {
			t.Errorf("%s: TypeTags() = '%s', want = '%s'", tt.desc, got, want)
		}
	}
}

func TestMessage_String(t *testing.T) {
	for _, tt := range []struct {
		desc string
		msg  *Message
		str  string
	}{
		{"nil", nil, ""},
		{"addr_only", NewMessage("/foo/bar"), "/foo/bar ,"},
		{"one_addr", NewMessage("/foo/bar", "123"), "/foo/bar ,s 123"},
		{"two_args", NewMessage("/foo/bar", "123", int32(456)), "/foo/bar ,si 123 456"},
	} {
		if got, want := tt.msg.String(), tt.str; got != want {
			t.Errorf("%s: String() = '%s', want = '%s'", tt.desc, got, want)
		}
	}
}

func TestHandle(t *testing.T) {
	server := &Server{Addr: "localhost:6677"}
	err := server.Handle("/address/test", func(msg *Message) {})
	if err != nil {
		t.Error("Expected that OSC address '/address/test' is valid")
	}
}

func TestHandleWithInvalidAddress(t *testing.T) {
	server := &Server{Addr: "localhost:6677"}
	err := server.Handle("/address*/test", func(msg *Message) {})
	if err == nil {
		t.Error("Expected error with '/address*/test'")
	}
}

func TestServerMessageDispatching(t *testing.T) {
	finish := make(chan bool)
	start := make(chan bool)
	done := sync.WaitGroup{}
	done.Add(2)

	// Start the OSC server in a new go-routine
	go func() {
		conn, err := net.ListenPacket("udp", "localhost:6677")
		if err != nil {
			t.Fatal(err)
		}
		defer conn.Close()

		server := &Server{Addr: "localhost:6677"}
		err = server.Handle("/address/test", func(msg *Message) {
			if len(msg.Arguments) != 1 {
				t.Error("Argument length should be 1 and is: " + string(len(msg.Arguments)))
			}

			if msg.Arguments[0].(int32) != 1122 {
				t.Error("Argument should be 1122 and is: " + string(msg.Arguments[0].(int32)))
			}

			// Stop OSC server
			conn.Close()
			finish <- true
		})
		if err != nil {
			t.Error("Error adding message handler")
		}

		start <- true
		server.Serve(conn)
	}()

	go func() {
		timeout := time.After(5 * time.Second)
		select {
		case <-timeout:
		case <-start:
			time.Sleep(500 * time.Millisecond)
			client := NewClient("localhost", 6677)
			msg := NewMessage("/address/test")
			msg.Append(int32(1122))
			client.Send(msg)
		}

		done.Done()

		select {
		case <-timeout:
		case <-finish:
		}
		done.Done()
	}()

	done.Wait()
}

func TestServerMessageReceiving(t *testing.T) {
	finish := make(chan bool)
	start := make(chan bool)
	done := sync.WaitGroup{}
	done.Add(2)

	// Start the server in a go-routine
	go func() {
		server := &Server{}
		c, err := net.ListenPacket("udp", "localhost:6677")
		if err != nil {
			t.Fatal(err)
		}
		defer c.Close()

		// Start the client
		start <- true

		packet, err := server.ReceivePacket(c)
		if err != nil {
			t.Error("Server error")
			return
		}
		if packet == nil {
			t.Error("nil packet")
			return
		}

		msg := packet.(*Message)
		if msg.CountArguments() != 2 {
			t.Errorf("Argument length should be 2 and is: %d\n", msg.CountArguments())
		}
		if msg.Arguments[0].(int32) != 1122 {
			t.Error("Argument should be 1122 and is: " + string(msg.Arguments[0].(int32)))
		}
		if msg.Arguments[1].(int32) != 3344 {
			t.Error("Argument should be 3344 and is: " + string(msg.Arguments[1].(int32)))
		}

		c.Close()
		finish <- true
	}()

	go func() {
		timeout := time.After(5 * time.Second)
		select {
		case <-timeout:
		case <-start:
			client := NewClient("localhost", 6677)
			msg := NewMessage("/address/test")
			msg.Append(int32(1122))
			msg.Append(int32(3344))
			time.Sleep(500 * time.Millisecond)
			client.Send(msg)
		}

		done.Done()

		select {
		case <-timeout:
		case <-finish:
		}
		done.Done()
	}()

	done.Wait()
}

func TestReadTimeout(t *testing.T) {
	start := make(chan bool)
	wg := sync.WaitGroup{}
	wg.Add(2)

	go func() {
		defer wg.Done()

		select {
		case <-time.After(5 * time.Second):
			t.Fatal("timed out")
		case <-start:
			client := NewClient("localhost", 6677)
			msg := NewMessage("/address/test1")
			err := client.Send(msg)
			if err != nil {
				t.Fatal(err)
			}
			time.Sleep(150 * time.Millisecond)
			msg = NewMessage("/address/test2")
			err = client.Send(msg)
			if err != nil {
				t.Fatal(err)
			}
		}
	}()

	go func() {
		defer wg.Done()

		server := &Server{ReadTimeout: 100 * time.Millisecond}
		c, err := net.ListenPacket("udp", "localhost:6677")
		if err != nil {
			t.Fatal(err)
		}
		defer c.Close()

		start <- true
		p, err := server.ReceivePacket(c)
		if err != nil {
			t.Errorf("server error: %v", err)
			return
		}
		if got, want := p.(*Message).Address, "/address/test1"; got != want {
			t.Errorf("wrong address; got = %s, want = %s", got, want)
			return
		}

		// Second receive should time out since client is delayed 150 milliseconds
		if _, err = server.ReceivePacket(c); err == nil {
			t.Errorf("expected error")
			return
		}

		// Next receive should get it
		p, err = server.ReceivePacket(c)
		if err != nil {
			t.Errorf("server error: %v", err)
			return
		}
		if got, want := p.(*Message).Address, "/address/test2"; got != want {
			t.Errorf("wrong address; got = %s, want = %s", got, want)
			return
		}
	}()

	wg.Wait()
}

func TestReadPaddedString(t *testing.T) {
	for _, tt := range []struct {
		buf []byte // buffer
		n   int    // bytes needed
		s   string // resulting string
	}{
		{[]byte{'t', 'e', 's', 't', 's', 't', 'r', 'i', 'n', 'g', 0, 0}, 12, "teststring"},
		{[]byte{'t', 'e', 's', 't', 0, 0, 0, 0}, 8, "test"},
	} {
		buf := bytes.NewBuffer(tt.buf)
		s, n, err := readPaddedString(bufio.NewReader(buf))
		if err != nil {
			t.Errorf("%s: Error reading padded string: %s", s, err)
		}
		if got, want := n, tt.n; got != want {
			t.Errorf("%s: Bytes needed don't match; got = %d, want = %d", tt.s, got, want)
		}
		if got, want := s, tt.s; got != want {
			t.Errorf("%s: Strings don't match; got = %d, want = %d", tt.s, got, want)
		}
	}
}

func TestWritePaddedString(t *testing.T) {
	buf := []byte{}
	bytesBuffer := bytes.NewBuffer(buf)
	testString := "testString"
	expectedNumberOfWrittenBytes := len(testString) + padBytesNeeded(len(testString))

	n, err := writePaddedString(testString, bytesBuffer)
	if err != nil {
		t.Errorf(err.Error())
	}

	if n != expectedNumberOfWrittenBytes {
		t.Errorf("Expected number of written bytes should be \"%d\" and is \"%d\"", expectedNumberOfWrittenBytes, n)
	}
}

func TestPadBytesNeeded(t *testing.T) {
	var n int
	n = padBytesNeeded(4)
	if n != 4 {
		t.Errorf("Number of pad bytes should be 4 and is: %d", n)
	}

	n = padBytesNeeded(3)
	if n != 1 {
		t.Errorf("Number of pad bytes should be 1 and is: %d", n)
	}

	n = padBytesNeeded(1)
	if n != 3 {
		t.Errorf("Number of pad bytes should be 3 and is: %d", n)
	}

	n = padBytesNeeded(0)
	if n != 4 {
		t.Errorf("Number of pad bytes should be 4 and is: %d", n)
	}

	n = padBytesNeeded(32)
	if n != 4 {
		t.Errorf("Number of pad bytes should be 4 and is: %d", n)
	}

	n = padBytesNeeded(63)
	if n != 1 {
		t.Errorf("Number of pad bytes should be 1 and is: %d", n)
	}

	n = padBytesNeeded(10)
	if n != 2 {
		t.Errorf("Number of pad bytes should be 2 and is: %d", n)
	}
}

func TestTypeTagsString(t *testing.T) {
	msg := NewMessage("/some/address")
	msg.Append(int32(100))
	msg.Append(true)
	msg.Append(false)

	typeTags, err := msg.TypeTags()
	if err != nil {
		t.Error(err.Error())
	}

	if typeTags != ",iTF" {
		t.Errorf("Type tag string should be ',iTF' and is: %s", typeTags)
	}
}

func TestClientSetLocalAddr(t *testing.T) {
	client := NewClient("localhost", 8967)
	err := client.SetLocalAddr("localhost", 41789)
	if err != nil {
		t.Error(err.Error())
	}
	expectedAddr := "127.0.0.1:41789"
	if client.laddr.String() != expectedAddr {
		t.Errorf("Expected laddr to be %s but was %s", expectedAddr, client.laddr.String())
	}
}

func TestParsePacket(t *testing.T) {
	for _, tt := range []struct {
		desc string
		msg  string
		pkt  Packet
		ok   bool
	}{
		{"no_args",
			"/a/b/c" + nulls(2) + "," + nulls(3),
			makePacket("/a/b/c", nil),
			true},
		{"string_arg",
			"/d/e/f" + nulls(2) + ",s" + nulls(2) + "foo" + nulls(1),
			makePacket("/d/e/f", []string{"foo"}),
			true},
		{"empty", "", nil, false},
	} {
		pkt, err := ParsePacket(tt.msg)
		if err != nil && tt.ok {
			t.Errorf("%s: ParsePacket() returned unexpected error; %s", tt.desc, err)
		}
		if err == nil && !tt.ok {
			t.Errorf("%s: ParsePacket() expected error", tt.desc)
		}
		if !tt.ok {
			continue
		}

		pktBytes, err := pkt.MarshalBinary()
		if err != nil {
			t.Errorf("%s: failure converting pkt to byte array; %s", tt.desc, err)
			continue
		}
		ttpktBytes, err := tt.pkt.MarshalBinary()
		if err != nil {
			t.Errorf("%s: failure converting tt.pkt to byte array; %s", tt.desc, err)
			continue
		}
		if got, want := pktBytes, ttpktBytes; !reflect.DeepEqual(got, want) {
			t.Errorf("%s: ParsePacket() as bytes = '%s', want = '%s'", tt.desc, got, want)
			continue
		}
	}
}

const zero = string(byte(0))

// nulls returns a string of `i` nulls.
func nulls(i int) string {
	s := ""
	for j := 0; j < i; j++ {
		s += zero
	}
	return s
}

// makePacket creates a fake Message Packet.
func makePacket(addr string, args []string) Packet {
	msg := NewMessage(addr)
	for _, arg := range args {
		msg.Append(arg)
	}
	return msg
}
