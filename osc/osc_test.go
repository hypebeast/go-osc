package osc

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net"
	"reflect"
	"strconv"
	"strings"
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
			t.Errorf("%s: TypeTags() expected an error", tt.desc)
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

func TestAddMsgHandler(t *testing.T) {
	d := NewStandardDispatcher()
	err := d.AddMsgHandler("/address/test", func(msg *Message) {})
	if err != nil {
		t.Error("Expected that OSC address '/address/test' is valid")
	}
}

func TestAddMsgHandlerWithInvalidAddress(t *testing.T) {
	d := NewStandardDispatcher()
	err := d.AddMsgHandler("/address*/test", func(msg *Message) {})
	if err == nil {
		t.Error("Expected error with '/address*/test'")
	}
}

// These tests stop the server by forcibly closing the connection, which causes
// a "use of closed network connection" error the next time we try to read from
// the connection. As a workaround, this wraps server.ListenAndServe() in an
// error-handling layer that doesn't consider "use of closed network connection"
// an error.
//
// Open question: is this desired behavior, or should server.serve return
// successfully in cases where it would otherwise throw this error?
func serveUntilInterrupted(server *Server) error {
	if err := server.ListenAndServe(); err != nil &&
		!strings.Contains(err.Error(), "use of closed network connection") {
		return err
	}

	return nil
}

func TestServerMessageDispatching(t *testing.T) {
	finish := make(chan bool)
	start := make(chan bool)
	done := sync.WaitGroup{}
	done.Add(2)

	port := 6677
	addr := "localhost:" + strconv.Itoa(port)

	d := NewStandardDispatcher()
	server := &Server{Addr: addr, Dispatcher: d}

	defer func() {
		if err := server.CloseConnection(); err != nil {
			t.Error(err)
		}
	}()

	if err := d.AddMsgHandler(
		"/address/test",
		func(msg *Message) {
			if len(msg.Arguments) != 1 {
				t.Errorf("Argument length should be 1 and is: %d", len(msg.Arguments))
			}

			if msg.Arguments[0].(int32) != 1122 {
				t.Error("Argument should be 1122 and is: " + string(msg.Arguments[0].(int32)))
			}

			finish <- true
		},
	); err != nil {
		t.Error("Error adding message handler")
	}

	// Server goroutine
	go func() {
		start <- true

		if err := serveUntilInterrupted(server); err != nil {
			t.Errorf("error during Serve: %s", err.Error())
		}
	}()

	// Client goroutine
	go func() {
		timeout := time.After(5 * time.Second)
		select {
		case <-timeout:
		case <-start:
			time.Sleep(500 * time.Millisecond)
			client := NewClient("localhost", port)
			msg := NewMessage("/address/test")
			msg.Append(int32(1122))
			if err := client.Send(msg); err != nil {
				t.Error(err)
				done.Done()
				done.Done()
				return
			}
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
	port := 6677

	finish := make(chan bool)
	start := make(chan bool)
	done := sync.WaitGroup{}
	done.Add(2)

	// Start the server in a go-routine
	go func() {
		server := &Server{}

		c, err := net.ListenPacket("udp", "localhost:"+strconv.Itoa(port))
		if err != nil {
			t.Error(err)
			return
		}
		defer c.Close()
		server.SetConnection(c)

		// Start the client
		start <- true

		packet, err := server.ReceivePacket()
		if err != nil {
			t.Errorf("server error: %v", err)
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
			client := NewClient("localhost", port)
			msg := NewMessage("/address/test")
			msg.Append(int32(1122))
			msg.Append(int32(3344))
			time.Sleep(500 * time.Millisecond)
			if err := client.Send(msg); err != nil {
				t.Error(err)
				done.Done()
				done.Done()
				return
			}
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
			t.Error("timed out")
			wg.Done()
		case <-start:
			client := NewClient("localhost", 6677)
			msg := NewMessage("/address/test1")
			err := client.Send(msg)
			if err != nil {
				t.Error(err)
			}

			time.Sleep(150 * time.Millisecond)
			msg = NewMessage("/address/test2")
			err = client.Send(msg)
			if err != nil {
				t.Error(err)
			}
		}
	}()

	go func() {
		defer wg.Done()

		server := &Server{ReadTimeout: 100 * time.Millisecond}
		c, err := net.ListenPacket("udp", "localhost:6677")
		if err != nil {
			t.Error(err)
		}
		defer c.Close()
		server.SetConnection(c)

		// Start the client
		start <- true
		p, err := server.ReceivePacket()
		if err != nil {
			t.Errorf("server error: %v", err)
			return
		}
		if got, want := p.(*Message).Address, "/address/test1"; got != want {
			t.Errorf("wrong address; got = %s, want = %s", got, want)
			return
		}

		// Second receive should time out since client is delayed 150 milliseconds
		if _, err = server.ReceivePacket(); err == nil {
			t.Errorf("expected error")
			return
		}

		// Next receive should get it
		p, err = server.ReceivePacket()
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

func TestReadBlob(t *testing.T) {
	for _, tt := range []struct {
		name    string
		args    []byte
		want    []byte
		want1   int
		wantErr bool
	}{
		{"negative value", []byte{255, 255, 255, 255}, nil, 0, true},
		{"large value", []byte{0, 1, 17, 112}, nil, 0, true},
		{"regular value", []byte{0, 0, 0, 1, 10, 0, 0, 0}, []byte{10}, 8, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := readBlob(bufio.NewReader(bytes.NewBuffer(tt.args)))
			if (err != nil) != tt.wantErr {
				t.Errorf("readBlob() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("readBlob() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("readBlob() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestReadPaddedString(t *testing.T) {
	for _, tt := range []struct {
		buf []byte // buffer
		n   int    // bytes needed
		s   string // resulting string
		e   error  // expected error
	}{
		{[]byte{'t', 'e', 's', 't', 'S', 't', 'r', 'i', 'n', 'g', 0, 0}, 12, "testString", nil},
		{[]byte{'t', 'e', 's', 't', 'e', 'r', 's', 0}, 8, "testers", nil},
		{[]byte{'t', 'e', 's', 't', 's', 0, 0, 0}, 8, "tests", nil},
		{[]byte{'t', 'e', 's', 't', 0, 0, 0, 0}, 8, "test", nil},
		{[]byte{}, 0, "", io.EOF},
		{[]byte{'t', 'e', 's', 0}, 4, "tes", nil},             // OSC uses null terminated strings
		{[]byte{'t', 'e', 's', 0, 0, 0, 0, 0}, 4, "tes", nil}, // Additional nulls should be ignored
		{[]byte{'t', 'e', 's', 0, 0, 0}, 4, "tes", nil},       // Whether or not the nulls fall on a 4 byte padding boundary
		{[]byte{'t', 'e', 's', 't'}, 0, "", io.EOF},           // if there is no null byte at the end, it doesn't work.
	} {
		buf := bytes.NewBuffer(tt.buf)
		s, n, err := readPaddedString(bufio.NewReader(buf))
		if got, want := err, tt.e; got != want {
			t.Errorf("%q: Unexpected error reading padded string; got = %q, want = %q", tt.s, got, want)
		}
		if got, want := n, tt.n; got != want {
			t.Errorf("%q: Bytes needed don't match; got = %d, want = %d", tt.s, got, want)
		}
		if got, want := s, tt.s; got != want {
			t.Errorf("%q: Strings don't match; got = %q, want = %q", tt.s, got, want)
		}
	}
}

func TestWritePaddedString(t *testing.T) {
	for _, tt := range []struct {
		s   string // string
		buf []byte // resulting buffer
		n   int    // bytes expected
	}{
		{"testString", []byte{'t', 'e', 's', 't', 'S', 't', 'r', 'i', 'n', 'g', 0, 0}, 12},
		{"testers", []byte{'t', 'e', 's', 't', 'e', 'r', 's', 0}, 8},
		{"tests", []byte{'t', 'e', 's', 't', 's', 0, 0, 0}, 8},
		{"test", []byte{'t', 'e', 's', 't', 0, 0, 0, 0}, 8},
		{"tes", []byte{'t', 'e', 's', 0}, 4},
		{"tes\x00", []byte{'t', 'e', 's', 0}, 4},                 // Don't add a second null terminator if one is already present
		{"tes\x00\x00\x00\x00\x00", []byte{'t', 'e', 's', 0}, 4}, // Skip extra nulls
		{"tes\x00\x00\x00", []byte{'t', 'e', 's', 0}, 4},         // Even if they don't fall on a 4 byte padding boundary
		{"", []byte{0, 0, 0, 0}, 4},                              // OSC uses null terminated strings, padded to the 4 byte boundary
	} {
		buf := []byte{}
		bytesBuffer := bytes.NewBuffer(buf)

		n, err := writePaddedString(tt.s, bytesBuffer)
		if err != nil {
			t.Errorf(err.Error())
		}
		if got, want := n, tt.n; got != want {
			t.Errorf("%q: Count of bytes written don't match; got = %d, want = %d", tt.s, got, want)
		}
		if got, want := bytesBuffer, tt.buf; !bytes.Equal(got.Bytes(), want) {
			t.Errorf("%q: Buffers don't match; got = %q, want = %q", tt.s, got.Bytes(), want)
		}
	}
}

func TestPadBytesNeeded(t *testing.T) {
	var n int
	n = padBytesNeeded(4)
	if n != 0 {
		t.Errorf("Number of pad bytes should be 0 and is: %d", n)
	}

	n = padBytesNeeded(3)
	if n != 1 {
		t.Errorf("Number of pad bytes should be 1 and is: %d", n)
	}

	n = padBytesNeeded(2)
	if n != 2 {
		t.Errorf("Number of pad bytes should be 2 and is: %d", n)
	}

	n = padBytesNeeded(1)
	if n != 3 {
		t.Errorf("Number of pad bytes should be 3 and is: %d", n)
	}

	n = padBytesNeeded(0)
	if n != 0 {
		t.Errorf("Number of pad bytes should be 0 and is: %d", n)
	}

	n = padBytesNeeded(5)
	if n != 3 {
		t.Errorf("Number of pad bytes should be 3 and is: %d", n)
	}

	n = padBytesNeeded(7)
	if n != 1 {
		t.Errorf("Number of pad bytes should be 1 and is: %d", n)
	}

	n = padBytesNeeded(32)
	if n != 0 {
		t.Errorf("Number of pad bytes should be 0 and is: %d", n)
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

func TestOscMessageMatch(t *testing.T) {
	tc := []struct {
		desc        string
		addr        string
		addrPattern string
		want        bool
	}{
		{
			"match everything",
			"*",
			"/a/b",
			true,
		},
		{
			"don't match",
			"/a/b",
			"/a",
			false,
		},
		{
			"match alternatives",
			"/a/{foo,bar}",
			"/a/foo",
			true,
		},
		{
			"don't match if address is not part of the alternatives",
			"/a/{foo,bar}",
			"/a/bob",
			false,
		},
	}

	for _, tt := range tc {
		msg := NewMessage(tt.addr)

		got := msg.Match(tt.addrPattern)
		if got != tt.want {
			t.Errorf("%s: msg.Match('%s') = '%t', want = '%t'", tt.desc, tt.addrPattern, got, tt.want)
		}
	}
}

func TestServerSend(t *testing.T) {
	targetServer := Server{
		Addr: "127.0.0.1:6677",
	}

	go func() {
		d := NewStandardDispatcher()
		d.AddMsgHandler("/message/test", func(msg *Message) {
			reply := NewMessage("/reply/test")
			err := targetServer.SendTo(reply, msg.SenderAddr())
			if err != nil {
				t.Errorf("SendTo failed: %v", err)
			}
		})
		targetServer.Dispatcher = d
		targetServer.ListenAndServe()
	}()

	time.Sleep(2 * time.Second)

	result := make(chan bool, 1)

	d := NewStandardDispatcher()
	d.AddMsgHandler("/reply/test", func(msg *Message) {
		result <- true
	})
	clientServer := Server{
		Addr:       "127.0.0.1:18536",
		Dispatcher: d,
	}

	err := clientServer.Listen()
	if err != nil {
		t.Errorf("Listen failed: %v", err)
	}
	go clientServer.Serve()

	msg := NewMessage("/message/test")
	addr, _ := net.ResolveUDPAddr("udp", targetServer.Addr)
	err = clientServer.SendTo(msg, addr)
	if err != nil {
		t.Errorf("SendTo failed: %v", err)
	}

	select {
	case r := <-result:
		if !r {
			t.Error("did not get expected response")
		}
	case <-time.After(2 * time.Second):
		t.Error("unexpected timeout")
	}
}

func TestClientRecv(t *testing.T) {
	targetServer := Server{
		Addr: "127.0.0.1:6678",
	}
	defer targetServer.CloseConnection()

	go func() {
		d := NewStandardDispatcher()
		d.AddMsgHandler("/message/test", func(msg *Message) {
			fmt.Println("DEBUG: targetServer: got /message/test")
			reply := NewMessage("/reply/test")
			err := targetServer.SendTo(reply, msg.SenderAddr())
			if err != nil {
				t.Errorf("SendTo failed: %v", err)
			}
		})
		targetServer.Dispatcher = d
		targetServer.ListenAndServe()
	}()

	time.Sleep(2 * time.Second)

	result := make(chan bool, 1)

	d := NewStandardDispatcher()
	d.AddMsgHandler("/reply/test", func(msg *Message) {
		result <- true
	})

	client := NewClient("127.0.0.1", 6677)
	client.SetDispatcher(d)
	go client.ListenAndServe()

	msg := NewMessage("/message/test")
	err := client.Send(msg)
	defer client.Close()
	if err != nil {
		t.Errorf("SendTo failed: %v", err)
	}

	select {
	case r := <-result:
		if !r {
			t.Error("did not get expected response")
		}
	case <-time.After(2 * time.Second):
		t.Error("unexpected timeout")
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
