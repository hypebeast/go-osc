package osc

import (
	"bufio"
	"bytes"
	"sync"
	"testing"
	"time"
)

func TestAppendArguments(t *testing.T) {
	oscAddress := "/address"
	message := NewOscMessage(oscAddress)
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

func TestEqualMessage(t *testing.T) {
	msg1 := NewOscMessage("/address")
	msg2 := NewOscMessage("/address")

	msg1.Append(1234)
	msg2.Append(1234)
	msg1.Append("test string")
	msg2.Append("test string")

	if !msg1.Equals(msg2) {
		t.Error("Messages should be equal")
	}
}

func TestAddMsgHandler(t *testing.T) {
	server := NewOscServer("localhost", 6677)
	err := server.AddMsgHandler("/address/test", func(msg *OscMessage) {})
	if err != nil {
		t.Error("Expected that OSC address '/address/test' is valid")
	}
}

func TestAddMsgHandlerWithInvalidAddress(t *testing.T) {
	server := NewOscServer("localhost", 6677)
	err := server.AddMsgHandler("/address*/test", func(msg *OscMessage) {})
	if err == nil {
		t.Error("Expected error with '/address*/test'")
	}
}

func TestServerMessageDispatching(t *testing.T) {
	finish := make(chan bool)
	start := make(chan bool)
	var done sync.WaitGroup
	done.Add(2)

	// Start the OSC server in a new go-routine
	go func() {
		server := NewOscServer("localhost", 6677)
		err := server.AddMsgHandler("/address/test", func(msg *OscMessage) {
			if len(msg.Arguments) != 1 {
				t.Error("Argument length should be 1 and is: " + string(len(msg.Arguments)))
			}

			if msg.Arguments[0].(int32) != 1122 {
				t.Error("Argument should be 1122 and is: " + string(msg.Arguments[0].(int32)))
			}

			// Stop OSC server
			server.Close()
			finish <- true
		})

		if err != nil {
			t.Error("Error adding message handler")
		}

		start <- true
		server.ListenAndDispatch()
	}()

	go func() {
		timeout := time.After(5 * time.Second)
		select {
		case <-timeout:
		case <-start:
			time.Sleep(500 * time.Millisecond)
			client := NewOscClient("localhost", 6677)
			msg := NewOscMessage("/address/test")
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
	var done sync.WaitGroup
	done.Add(2)

	// Start the server in a go-routine
	go func() {
		server := NewOscServer("localhost", 6677)
		server.Listen()

		// Start the client
		start <- true

		for {
			packet, err := server.ReceivePacket()
			if err != nil {
				t.Error("Server error")
			}

			if packet != nil {
				msg := packet.(*OscMessage)
				if msg.CountArguments() != 2 {
					t.Errorf("Argument length should be 2 and is: %d\n", msg.CountArguments())
				}

				if msg.Arguments[0].(int32) != 1122 {
					t.Error("Argument should be 1122 and is: " + string(msg.Arguments[0].(int32)))
				}

				if msg.Arguments[1].(int32) != 3344 {
					t.Error("Argument should be 3344 and is: " + string(msg.Arguments[1].(int32)))
				}

				server.Close()
				finish <- true
			}
		}
	}()

	go func() {
		timeout := time.After(5 * time.Second)
		select {
		case <-timeout:
		case <-start:
			client := NewOscClient("localhost", 6677)
			msg := NewOscMessage("/address/test")
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

func TestReadPaddedString(t *testing.T) {
	buf1 := []byte{'t', 'e', 's', 't', 's', 't', 'r', 'i', 'n', 'g', 0, 0}
	buf2 := []byte{'t', 'e', 's', 't', 0, 0, 0, 0}

	bytesBuffer := bytes.NewBuffer(buf1)
	st, n, err := readPaddedString(bufio.NewReader(bytesBuffer))
	if err != nil {
		t.Error("Error reading padded string: " + err.Error())
	}

	if n != 12 {
		t.Errorf("Number of bytes needs to be 12 and is: %d\n", n)
	}

	if st != "teststring" {
		t.Errorf("String should be \"teststring\" and is \"%s\"", st)
	}

	bytesBuffer = bytes.NewBuffer(buf2)
	st, n, err = readPaddedString(bufio.NewReader(bytesBuffer))
	if err != nil {
		t.Error("Error reading padded string: " + err.Error())
	}

	if n != 8 {
		t.Errorf("Number of bytes needs to be 8 and is: %d\n", n)
	}

	if st != "test" {
		t.Errorf("String should be \"test\" and is \"%s\"", st)
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
	msg := NewOscMessage("/some/address")
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

func TestServerIsNotRunningAndGetsClosed(t *testing.T) {
	server := NewOscServer("127.0.0.1", 8000)
	err := server.Close()
	if err == nil {
		t.Errorf("Expected error if the the server is not running and it gets closed")
	}
}
