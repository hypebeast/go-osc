// go-osc provides a package for sending and receiving OpenSoundControl messages.
// The package is implemented in pure Go.
package osc

import (
	"bufio"
	"bytes"
	"encoding"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"regexp"
	"strings"
	"time"
)

const (
	secondsFrom1900To1970 = 2208988800
	bundleTagString       = "#bundle"
)

// Packet is the interface for Message and Bundle.
type Packet interface {
	encoding.BinaryMarshaler
}

// Message represents a single OSC message. An OSC message consists of an OSC address
// pattern and zero or more arguments.
type Message struct {
	Address   string
	Arguments []interface{}
}

// Bundle represents an OSC bundle. It consists of the OSC-string "#bundle" followed
// by an OSC Time Tag, followed by zero or more OSC bundle/message elements. The
// OSC-timetag is a 64-bit fixed point time tag. See
// http://opensoundcontrol.org/spec-1_0 for more information.
type Bundle struct {
	Timetag  Timetag
	Messages []*Message
	Bundles  []*Bundle
}

// Client enables you to send OSC packets. It sends OSC messages and bundles to the
// given IP address and port.
type Client struct {
	ipaddress string
	port      int
	laddr     *net.UDPAddr
}

// Server represents an OSC server. The server listens on Address and Port for
// incoming OSC packets and bundles.
type Server struct {
	Addr        string
	Dispatcher  *OscDispatcher
	ReadTimeout time.Duration
}

// Timetag represents an OSC Time Tag.
// An OSC Time Tag is defined as follows:
// Time tags are represented by a 64 bit fixed point number. The first 32 bits
// specify the number of seconds since midnight on January 1, 1900, and the
// last 32 bits specify fractional parts of a second to a precision of about
// 200 picoseconds. This is the representation used by Internet NTP timestamps.
type Timetag struct {
	timeTag  uint64 // The acutal time tag
	time     time.Time
	MinValue uint64 // Minimum value of an OSC Time Tag. Is always 1.
}

// Dispatcher is an interface for an OSC message dispatcher. A dispatcher is
// responsible for dispatching received OSC messages.
type Dispatcher interface {
	Dispatch(packet Packet)
}

// Handler is an interface for message handlers. Every handler implementation for
// an OSC message must implement this interface.
type Handler interface {
	HandleMessage(msg *Message)
}

// HandlerFunc implements the Handler interface. Type defintion for an OSC handler function
type HandlerFunc func(msg *Message)

// HandleMessage calls themeself with the given OSC Message. Implements the
// Handler interface.
func (f HandlerFunc) HandleMessage(msg *Message) {
	f(msg)
}

////
// OscDispatcher
////

// OscDispatcher is a dispatcher for OSC packets. It handles the dispatching of
// received OSC packets.
type OscDispatcher struct {
	handlers       map[string]Handler
	defaultHandler Handler
}

// NewOscDispatcher returns an OscDispatcher.
func NewOscDispatcher() (dispatcher *OscDispatcher) {
	return &OscDispatcher{handlers: make(map[string]Handler)}
}

// AddMsgHandler adds a new message handler for the given OSC address.
func (s *OscDispatcher) AddMsgHandler(address string, handler HandlerFunc) error {
	if address == "*" {
		s.defaultHandler = handler
		return nil
	}
	for _, chr := range "*?,[]{}# " {
		if strings.Contains(address, fmt.Sprintf("%c", chr)) {
			return errors.New("OSC Address string may not contain any characters in \"*?,[]{}# \n")
		}
	}

	if existsAddress(address, s.handlers) {
		return errors.New("OSC address exists already")
	}

	s.handlers[address] = handler

	return nil
}

// Dispatch dispatches OSC packets. Implements the Dispatcher interface.
func (s *OscDispatcher) Dispatch(packet Packet) {
	switch packet.(type) {
	default:
		return

	case *Message:
		msg, _ := packet.(*Message)
		for address, handler := range s.handlers {
			if msg.Match(address) {
				handler.HandleMessage(msg)
			}
		}
		if s.defaultHandler != nil {
			s.defaultHandler.HandleMessage(msg)
		}

	case *Bundle:
		bundle, _ := packet.(*Bundle)
		timer := time.NewTimer(bundle.Timetag.ExpiresIn())

		go func() {
			<-timer.C
			for _, message := range bundle.Messages {
				for address, handler := range s.handlers {
					if message.Match(address) {
						handler.HandleMessage(message)
					}
				}
				if s.defaultHandler != nil {
					s.defaultHandler.HandleMessage(message)
				}
			}

			// Process all bundles
			for _, b := range bundle.Bundles {
				s.Dispatch(b)
			}
		}()
	}
}

////
// Message
////

// NewMessage returns a new Message. The address parameter is the OSC address.
func NewMessage(address string, arguments ...interface{}) (msg *Message) {
	return &Message{Address: address, Arguments: arguments}
}

// Append appends the given argument to the arguments list.
func (msg *Message) Append(arguments ...interface{}) {
	msg.Arguments = append(msg.Arguments, arguments...)
}

// Equals determines if the given OSC Message b is equal to the current OSC Message.
// It checks if the OSC address and the arguments are equal. Returns, true if the
// current object and b are equal.
func (msg *Message) Equals(b *Message) bool {
	// Check OSC address
	if msg.Address != b.Address {
		return false
	}

	// Check if the number of arguments are equal
	if msg.CountArguments() != b.CountArguments() {
		return false
	}

	// Check arguments
	for i, arg := range msg.Arguments {
		switch arg.(type) {
		case bool, int32, int64, float32, float64, string:
			if arg != b.Arguments[i] {
				return false
			}

		case []byte:
			ba := arg.([]byte)
			bb := b.Arguments[i].([]byte)
			if !bytes.Equal(ba, bb) {
				return false
			}

		case Timetag:
			if arg.(*Timetag).TimeTag() != b.Arguments[i].(*Timetag).TimeTag() {
				return false
			}
		}
	}

	return true
}

// Clear clears the OSC address and all arguments.
func (msg *Message) Clear() {
	msg.Address = ""
	msg.ClearData()
}

// ClearData removes all arguments from the OSC Message.
func (msg *Message) ClearData() {
	msg.Arguments = msg.Arguments[len(msg.Arguments):]
}

// Match returns true, if the address of the OSC Message matches the given address.
// The match is case sensitive!
func (msg *Message) Match(address string) bool {
	exp := getRegEx(msg.Address)

	if exp.MatchString(address) {
		return true
	}

	return false
}

// TypeTags returns the type tag string.
func (msg *Message) TypeTags() (tags string, err error) {
	tags = ","
	for _, m := range msg.Arguments {
		s, err := getTypeTag(m)
		if err != nil {
			return "", err
		}
		tags += s
	}

	return tags, nil
}

func (msg *Message) String() string {
	tags, err := msg.TypeTags()
	if err != nil {
		return ""
	}

	var formatString string
	var arguments []interface{}
	formatString += "%s %s"
	arguments = append(arguments, msg.Address)
	arguments = append(arguments, tags)

	for _, arg := range msg.Arguments {
		switch arg.(type) {
		case bool, int32, int64, float32, float64, string:
			formatString += " %v"
			arguments = append(arguments, arg)

		case nil:
			formatString += " %s"
			arguments = append(arguments, "Nil")

		case []byte:
			formatString += " %s"
			arguments = append(arguments, "blob")

		case Timetag:
			formatString += " %d"
			timeTag := arg.(Timetag)
			arguments = append(arguments, timeTag.TimeTag())
		}
	}
	return fmt.Sprintf(formatString, arguments...)
}

// CountArguments returns the number of arguments.
func (msg *Message) CountArguments() int {
	return len(msg.Arguments)
}

// MarshalBinary serializes the OSC message to a byte buffer. The byte buffer
// has the following format:
// 1. OSC Address Pattern
// 2. OSC Type Tag String
// 3. OSC Arguments
func (msg *Message) MarshalBinary() (buffer []byte, err error) {
	// The byte buffer for the message
	var data = new(bytes.Buffer)

	// We can start with the OSC address and add it to the buffer
	_, err = writePaddedString(msg.Address, data)
	if err != nil {
		return nil, err
	}

	// Type tag string starts with ","
	typetags := []byte{','}

	// Process the type tags and collect all arguments
	var payload = new(bytes.Buffer)
	for _, arg := range msg.Arguments {
		// FIXME: Use t instead of arg
		switch t := arg.(type) {
		default:
			return nil, fmt.Errorf("OSC - unsupported type: %T", t)

		case bool:
			if arg.(bool) == true {
				typetags = append(typetags, 'T')
			} else {
				typetags = append(typetags, 'F')
			}

		case nil:
			typetags = append(typetags, 'N')

		case int32:
			typetags = append(typetags, 'i')

			if err = binary.Write(payload, binary.BigEndian, int32(t)); err != nil {
				return nil, err
			}

		case float32:
			typetags = append(typetags, 'f')

			if err = binary.Write(payload, binary.BigEndian, float32(t)); err != nil {
				return nil, err
			}

		case string:
			typetags = append(typetags, 's')

			if _, err = writePaddedString(t, payload); err != nil {
				return nil, err
			}

		case []byte:
			typetags = append(typetags, 'b')

			if _, err = writeBlob(t, payload); err != nil {
				return nil, err
			}

		case int64:
			typetags = append(typetags, 'h')

			if err = binary.Write(payload, binary.BigEndian, int64(t)); err != nil {
				return nil, err
			}

		case float64:
			typetags = append(typetags, 'd')

			if err = binary.Write(payload, binary.BigEndian, float64(t)); err != nil {
				return nil, err
			}

		case Timetag:
			typetags = append(typetags, 't')

			timeTag := arg.(Timetag)
			b, _ := timeTag.MarshalBinary()
			_, err = payload.Write(b)
			if err != nil {
				return nil, err
			}
		}
	}

	// Write the type tag string to the data buffer
	_, err = writePaddedString(string(typetags), data)
	if err != nil {
		return nil, err
	}

	// Write the payload (OSC arguments) to the data buffer
	_, err = data.Write(payload.Bytes())
	if err != nil {
		return nil, err
	}

	return data.Bytes(), nil
}

////
// Bundle
////

// NewBundle returns an OSC Bundle. Use this function to create a new OSC
// Bundle.
func NewBundle(time time.Time) (bundle *Bundle) {
	return &Bundle{Timetag: *NewTimetag(time)}
}

// Append appends an OSC bundle or OSC message to the bundle.
func (b *Bundle) Append(pck Packet) (err error) {
	switch t := pck.(type) {
	default:
		return fmt.Errorf("Unsupported OSC packet type: only Bundle and Message are supported.")

	case *Bundle:
		b.Bundles = append(b.Bundles, t)

	case *Message:
		b.Messages = append(b.Messages, t)
	}

	return nil
}

// MarshalBinary serializes the OSC bundle to a byte array with the following format:
// 1. Bundle string: '#bundle'
// 2. OSC timetag
// 3. Length of first OSC bundle element
// 4. First bundle element
// 5. Length of n OSC bundle element
// 6. n bundle element
func (b *Bundle) MarshalBinary() (buffer []byte, err error) {
	var data = new(bytes.Buffer)

	// Add the '#bundle' string
	_, err = writePaddedString("#bundle", data)
	if err != nil {
		return nil, err
	}

	// Add the timetag
	bd, _ := b.Timetag.MarshalBinary()
	if _, err = data.Write(bd); err != nil {
		return nil, err
	}

	// Process all OSC Messages
	for _, m := range b.Messages {
		var msgLen int
		var msgBuf []byte

		msgBuf, err = m.MarshalBinary()
		if err != nil {
			return nil, err
		}

		// Append the length of the OSC message
		msgLen = len(msgBuf)
		if err = binary.Write(data, binary.BigEndian, int32(msgLen)); err != nil {
			return nil, err
		}

		// Append the OSC message
		_, err = data.Write(msgBuf)
		if err != nil {
			return nil, err
		}
	}

	// Process all OSC Bundles
	for _, b := range b.Bundles {
		var bLen int
		var bBuf []byte

		bBuf, err = b.MarshalBinary()
		if err != nil {
			return nil, err
		}

		// Write the size of the bundle
		bLen = len(bBuf)
		if err = binary.Write(data, binary.BigEndian, int32(bLen)); err != nil {
			return nil, err
		}

		// Append the bundle
		_, err = data.Write(bBuf)
		if err != nil {
			return nil, err
		}
	}

	return data.Bytes(), nil
}

////
// Client
////

// NewClient creates a new OSC client. The Client is used to send OSC
// messages and OSC bundles over an UDP network connection. The argument ip
// specifies the IP address and port defines the target port where the messages
// and bundles will be send to.
func NewClient(ip string, port int) (client *Client) {
	return &Client{ipaddress: ip, port: port, laddr: nil}
}

// Ip returns the IP address.
func (client *Client) IP() string {
	return client.ipaddress
}

// SetIp sets a new IP address.
func (client *Client) SetIP(ip string) {
	client.ipaddress = ip
}

// Port returns the port.
func (client *Client) Port() int {
	return client.port
}

// SetPort sets a new port.
func (client *Client) SetPort(port int) {
	client.port = port
}

// SetLocalAddr sets the local address.
func (client *Client) SetLocalAddr(ip string, port int) error {
	laddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", ip, port))
	if err != nil {
		return err
	}
	client.laddr = laddr
	return nil
}

// Send sends an OSC Bundle or an OSC Message.
func (client *Client) Send(packet Packet) (err error) {
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", client.ipaddress, client.port))
	conn, err := net.DialUDP("udp", client.laddr, addr)
	if err != nil {
		return err
	}
	defer conn.Close()

	data, err := packet.MarshalBinary()
	if err != nil {
		return err
	}

	_, err = conn.Write(data)
	if err != nil {
		return err
	}

	return nil
}

////
// Server
////

// Handle registers a new message handler function for an OSC address. The handler
// is the function called for incoming OscMessages that match 'address'.
func (s *Server) Handle(address string, handler HandlerFunc) error {
	if s.Dispatcher == nil {
		s.Dispatcher = NewOscDispatcher()
	}
	return s.Dispatcher.AddMsgHandler(address, handler)
}

// ListenAndServe retrieves incoming OSC packets and dispatches the retrieved
// OSC packets.
func (s *Server) ListenAndServe() error {
	if s.Dispatcher == nil {
		s.Dispatcher = NewOscDispatcher()
	}

	ln, err := net.ListenPacket("udp", s.Addr)
	if err != nil {
		return err
	}
	return s.Serve(ln)
}

// Serve retrieves incoming OSC packets from the given connection and dispatches
// retreived OSC packets. If something goes wrong an error is returned.
func (s *Server) Serve(c net.PacketConn) error {
	var tempDelay time.Duration
	for {
		msg, err := s.readFromConnection(c)
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}
				if max := 1 * time.Second; tempDelay > max {
					tempDelay = max
				}
				time.Sleep(tempDelay)
				continue
			}
			return err
		}
		tempDelay = 0
		go s.Dispatcher.Dispatch(msg)
	}

	return nil
}

// ReceivePacket listens for incoming OSC packets and returns the packet if one is received.
func (s *Server) ReceivePacket(c net.PacketConn) (packet Packet, err error) {
	return s.readFromConnection(c)
}

// readFromConnection retrieves OSC packets.
func (s *Server) readFromConnection(c net.PacketConn) (packet Packet, err error) {
	if s.ReadTimeout != 0 {
		err = c.SetReadDeadline(time.Now().Add(s.ReadTimeout))
		if err != nil {
			return nil, err
		}
	}
	data := make([]byte, 65535)
	var n, start int
	n, _, err = c.ReadFrom(data)
	if err != nil {
		return nil, err
	}
	return readPacket(bufio.NewReader(bytes.NewBuffer(data)), &start, n)
}

func ParsePacket(msgString string) (packet Packet, err error) {
	var start int
	return readPacket(bufio.NewReader(bytes.NewBufferString(msgString)), &start, len(msgString))
}


// receivePacket receives an OSC packet from the given reader.
func readPacket(reader *bufio.Reader, start *int, end int) (packet Packet, err error) {
	var buf []byte
	buf, err = reader.Peek(1)
	if err != nil {
		return nil, err
	}

	// An OSC Message starts with a '/'
	if buf[0] == '/' {
		packet, err = readMessage(reader, start)
		if err != nil {
			return nil, err
		}
	} else if buf[0] == '#' { // An OSC bundle starts with a '#'
		packet, err = readBundle(reader, start, end)
		if err != nil {
			return nil, err
		}
	}

	return packet, nil
}

// readBundle reads an Bundle from reader.
func readBundle(reader *bufio.Reader, start *int, end int) (bundle *Bundle, err error) {
	// Read the '#bundle' OSC string
	var startTag string
	var n int
	startTag, n, err = readPaddedString(reader)
	if err != nil {
		return nil, err
	}
	*start += n

	if startTag != bundleTagString {
		return nil, fmt.Errorf("Invalid bundle start tag: %s", startTag)
	}

	// Read the timetag
	var timeTag uint64
	if err := binary.Read(reader, binary.BigEndian, &timeTag); err != nil {
		return nil, err
	}
	*start += 8

	// Create a new bundle
	bundle = NewBundle(timetagToTime(timeTag))

	// Read until the end of the buffer
	for *start < end {
		// Read the size of the bundle element
		var length int32
		err = binary.Read(reader, binary.BigEndian, &length)
		*start += 4
		if err != nil {
			return nil, err
		}

		var packet Packet
		packet, err = readPacket(reader, start, end)
		if err != nil {
			return nil, err
		}
		err = bundle.Append(packet)
		if err != nil {
			return nil, err
		}
	}

	return bundle, nil
}

// readMessage reads one OSC Message from reader.
func readMessage(reader *bufio.Reader, start *int) (msg *Message, err error) {
	// First, read the OSC address
	var n int
	address, n, err := readPaddedString(reader)
	if err != nil {
		return nil, err
	}
	*start += n

	// Create a new message
	msg = NewMessage(address)

	// Read all arguments
	if err = readArguments(msg, reader, start); err != nil {
		return nil, err
	}

	return msg, nil
}

// readArguments reads all arguments from the reader and adds it to the OSC message.
func readArguments(msg *Message, reader *bufio.Reader, start *int) error {
	// Read the type tag string
	var n int
	typetags, n, err := readPaddedString(reader)
	if err != nil {
		return err
	}
	*start += n

	// If the typetag doesn't start with ',', it's not valid
	if typetags[0] != ',' {
		return errors.New("unsupported type tag string")
	}

	// Remove ',' from the type tag
	typetags = typetags[1:]

	for _, c := range typetags {
		switch c {
		default:
			return fmt.Errorf("unsupported type tag: %c", c)

		// int32
		case 'i':
			var i int32
			if err = binary.Read(reader, binary.BigEndian, &i); err != nil {
				return err
			}
			*start += 4
			msg.Append(i)

		// int64
		case 'h':
			var i int64
			if err = binary.Read(reader, binary.BigEndian, &i); err != nil {
				return err
			}
			*start += 8
			msg.Append(i)

		// float32
		case 'f':
			var f float32
			if err = binary.Read(reader, binary.BigEndian, &f); err != nil {
				return err
			}
			*start += 4
			msg.Append(f)

		// float64/double
		case 'd':
			var d float64
			if err = binary.Read(reader, binary.BigEndian, &d); err != nil {
				return err
			}
			*start += 8
			msg.Append(d)

		// string
		case 's':
			// TODO: fix reading string value
			var s string
			if s, _, err = readPaddedString(reader); err != nil {
				return err
			}
			*start += len(s) + padBytesNeeded(len(s))
			msg.Append(s)

		// blob
		case 'b':
			var buf []byte
			var n int
			if buf, n, err = readBlob(reader); err != nil {
				return err
			}
			*start += n
			msg.Append(buf)

		// OSC Time Tag
		case 't':
			var tt uint64
			if err = binary.Read(reader, binary.BigEndian, &tt); err != nil {
				return nil
			}
			*start += 8
			msg.Append(NewTimetagFromTimetag(tt))

		// True
		case 'T':
			msg.Append(true)

		// False
		case 'F':
			msg.Append(false)
		}
	}

	return nil
}

////
// Timetag
////

// NewTimetag returns a new OSC timetag object.
func NewTimetag(timeStamp time.Time) (timetag *Timetag) {
	return &Timetag{
		time:     timeStamp,
		timeTag:  timeToTimetag(timeStamp),
		MinValue: uint64(1)}
}

// NewTimetagFromTimetag creates a new Timetag from the given time tag.
func NewTimetagFromTimetag(timetag uint64) (t *Timetag) {
	time := timetagToTime(timetag)
	return NewTimetag(time)
}

// Time returns the time.
func (t *Timetag) Time() time.Time {
	return t.time
}

// FractionalSecond returns the last 32 bits of the Osc Time Tag. Specifies the
// fractional part of a second.
func (t *Timetag) FractionalSecond() uint32 {
	return uint32(t.timeTag << 32)
}

// SecondsSinceEpoch returns the first 32 bits (the number of seconds since the
// midnight 1900) from the OSC timetag.
func (t *Timetag) SecondsSinceEpoch() uint32 {
	return uint32(t.timeTag >> 32)
}

// TimeTag returns the time tag value
func (t *Timetag) TimeTag() uint64 {
	return t.timeTag
}

// MarshalBinary converts the OSC Time Tag to a byte array.
func (t *Timetag) MarshalBinary() ([]byte, error) {
	var data = new(bytes.Buffer)
	binary.Write(data, binary.BigEndian, t.timeTag)
	return data.Bytes(), nil
}

// SetTime sets the value of the OSC Time Tag.
func (t *Timetag) SetTime(time time.Time) {
	t.time = time
	t.timeTag = timeToTimetag(time)
}

// ExpiresIn calculates the number of seconds until the current time is the
// same as the value of the timetag. It returns zero if the value of the
// timetag is in the past.
func (t *Timetag) ExpiresIn() time.Duration {
	if t.timeTag <= 1 {
		return 0
	}

	tt := timetagToTime(t.timeTag)
	seconds := tt.Sub(time.Now())

	if seconds <= 0 {
		return 0
	}

	return seconds
}

////
// De/Encoding functions
////

// readBlob reads an OSC Blob from the blob byte array. Padding bytes are removed
// from the reader and not returned.
func readBlob(reader *bufio.Reader) (blob []byte, n int, err error) {
	// First, get the length
	var blobLen int
	if err = binary.Read(reader, binary.BigEndian, &blobLen); err != nil {
		return nil, 0, err
	}
	n = 4 + blobLen

	// Read the data
	blob = make([]byte, blobLen)
	if _, err = reader.Read(blob); err != nil {
		return nil, 0, err
	}

	// Remove the padding bytes
	numPadBytes := padBytesNeeded(blobLen)
	if numPadBytes > 0 {
		n += numPadBytes
		dummy := make([]byte, numPadBytes)
		if _, err = reader.Read(dummy); err != nil {
			return nil, 0, err
		}
	}

	return blob, n, nil
}

// writeBlob writes the data byte array as an OSC blob into buff. If the length of
// data isn't 32-bit aligned, padding bytes will be added.
func writeBlob(data []byte, buff *bytes.Buffer) (numberOfBytes int, err error) {
	// Add the size of the blob
	dlen := int32(len(data))
	err = binary.Write(buff, binary.BigEndian, dlen)
	if err != nil {
		return 0, err
	}

	// Write the data
	if _, err = buff.Write(data); err != nil {
		return 0, nil
	}

	// Add padding bytes if necessary
	numPadBytes := padBytesNeeded(len(data))
	if numPadBytes > 0 {
		padBytes := make([]byte, numPadBytes)
		if numPadBytes, err = buff.Write(padBytes); err != nil {
			return 0, err
		}
	}

	return 4 + len(data) + numPadBytes, nil
}

// readPaddedString reads a padded string from the given reader. The padding bytes
// are removed from the reader.
func readPaddedString(reader *bufio.Reader) (str string, n int, err error) {
	// Read the string from the reader
	str, err = reader.ReadString(0)
	if err != nil {
		return "", 0, err
	}
	n = len(str)

	// Remove the string delimiter, in order to calculate the right amount
	// of padding bytes
	str = str[:len(str)-1]

	// Remove the padding bytes
	padLen := padBytesNeeded(len(str)) - 1
	if padLen > 0 {
		n += padLen
		padBytes := make([]byte, padLen)
		if _, err = reader.Read(padBytes); err != nil {
			return "", 0, err
		}
	}

	return str, n, nil
}

// writePaddedString writes a string with padding bytes to the a buffer.
// Returns, the number of written bytes and an error if any.
func writePaddedString(str string, buff *bytes.Buffer) (numberOfBytes int, err error) {
	// Write the string to the buffer
	n, err := buff.WriteString(str)
	if err != nil {
		return 0, err
	}

	// Calculate the padding bytes needed and create a buffer for the padding bytes
	numPadBytes := padBytesNeeded(len(str))
	if numPadBytes > 0 {
		padBytes := make([]byte, numPadBytes)
		// Add the padding bytes to the buffer
		if numPadBytes, err = buff.Write(padBytes); err != nil {
			return 0, err
		}
	}

	return n + numPadBytes, nil
}

// padBytesNeeded determines how many bytes are needed to fill up to the next 4
// byte length.
func padBytesNeeded(elementLen int) int {
	return 4*(elementLen/4+1) - elementLen
}

////
// Timetag utility functions
////

// timeToTimetag converts the given time to an OSC timetag.
//
// An OSC timetage is defined as follows:
// Time tags are represented by a 64 bit fixed point number. The first 32 bits
// specify the number of seconds since midnight on January 1, 1900, and the
// last 32 bits specify fractional parts of a second to a precision of about
// 200 picoseconds. This is the representation used by Internet NTP timestamps.
//
// The time tag value consisting of 63 zero bits followed by a one in the least
// signifigant bit is a special case meaning "immediately."
func timeToTimetag(time time.Time) (timetag uint64) {
	timetag = uint64((secondsFrom1900To1970 + time.Unix()) << 32)
	return (timetag + uint64(uint32(time.Nanosecond())))
}

// timetagToTime converts the given timetag to a time object.
func timetagToTime(timetag uint64) (t time.Time) {
	return time.Unix(int64((timetag>>32)-secondsFrom1900To1970), int64(timetag&0xffffffff))
}

////
// Utility and helper functions
////

// PrintMessage pretty prints an OSC message to the standard output.
func PrintMessage(msg *Message) {
	fmt.Println(msg.String())
}

// existsAddress returns true if the OSC address s is found in handlers. Otherwise, false.
func existsAddress(s string, handlers map[string]Handler) bool {
	for address := range handlers {
		if address == s {
			return true
		}
	}

	return false
}

// getRegEx compiles and returns a regular expression object for the given address
// pattern.
func getRegEx(pattern string) *regexp.Regexp {
	pattern = strings.Replace(pattern, ".", "\\.", -1) // Escape all '.' in the pattern
	pattern = strings.Replace(pattern, "(", "\\(", -1) // Escape all '(' in the pattern
	pattern = strings.Replace(pattern, ")", "\\)", -1) // Escape all ')' in the pattern
	pattern = strings.Replace(pattern, "*", ".*", -1)  // Replace a '*' with '.*' that matches zero or more characters
	pattern = strings.Replace(pattern, "{", "(", -1)   // Change a '{' to '('
	pattern = strings.Replace(pattern, ",", "|", -1)   // Change a ',' to '|'
	pattern = strings.Replace(pattern, "}", ")", -1)   // Change a '}' to ')'
	pattern = strings.Replace(pattern, "?", ".", -1)   // Change a '?' to '.'

	return regexp.MustCompile(pattern)
}

// getTypeTag returns the OSC type tag for the given argument.
func getTypeTag(arg interface{}) (s string, err error) {
	switch t := arg.(type) {
	default:
		return "", fmt.Errorf("Unsupported type: %T", t)

	case bool:
		if arg.(bool) {
			s = "T"
		} else {
			s = "F"
		}

	case nil:
		s = "N"

	case int32:
		s = "i"

	case float32:
		s = "f"

	case string:
		s = "s"

	case []byte:
		s = "b"

	case int64:
		s = "h"

	case float64:
		s = "d"

	case Timetag:
		s = "t"
	}
	return s, nil
}
