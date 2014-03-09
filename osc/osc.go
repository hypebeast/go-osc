/*
 * go-osc provides a package for sending and receiving OpenSoundControl messages.
 * The package is implemented in pure Go.
 *
 * The implementation is based on the Open Sound Control 1.0 Specification
 * (http://opensoundcontrol.org/spec-1_0).
 *
 * Open Sound Control (OSC) is an open, transport-independent, message-based
 * protocol developed for communication among computers, sound synthesizers,
 * and other multimedia devices.
 *
 * This OSC implementation uses the UDP protocol for sending and receiving
 * OSC packets.
 *
 * The unit of transmission of OSC is an OSC Packet. Any application that sends
 * OSC Packets is an OSC Client; any application that receives OSC Packets is
 * an OSC Server.
 *
 * An OSC packet consists of its contents, a contiguous block of binary data,
 * and its size, the number of 8-bit bytes that comprise the contents. The
 * size of an OSC packet is always a multiple of 4.
 *
 * OSC packets come in two flavors:
 *
 * OSC Messages: An OSC message consists of an OSC address pattern, followed
 * by an OSC Type Tag String, and finally by zero or more OSC arguments.
 *
 * OSC Bundles: An OSC Bundle consists of the string "#bundle" followed
 * by an OSC Time Tag, followed by zero or more OSC bundle elements. Each bundle
 * element can be another OSC bundle (note this recursive definition: bundle may
 * contain bundles) or OSC message.
 *
 * An OSC bundle element consists of its size and its contents. The size is
 * an int32 representing the number of 8-bit bytes in the contents, and will
 * always be a multiple of 4. The contents are either an OSC Message or an
 * OSC Bundle.
 *
 * The following argument types are supported: 'i' (Int32), 'f' (Float32),
 * 's' (string), 'b' (blob / binary data), 'h' (Int64), 't' (OSC timetag),
 * 'd' (Double), 'r' (RGBA color), 'T' (True), 'F' (False), 'N' (Nil).
 *
 * Supported OSC address patterns: TODO: Add the supported address patterns
 *
 * TODO: Describe OscClient and OscServer
 *
 * Example usage:
 *
 * TODO: add some example usages
 *
 * Author: Sebastian Ruml <sebastian.ruml@gmail.com>
 * Created: 2013.08.19
 *
 */

package osc

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"regexp"
	"strings"
	"time"
)

const (
	// The time tag value consisting of 63 zero bits followed by a one in the
	// least signifigant bit is a special case meaning "immediately."
	timeTagImmediate      = uint64(1)
	secondsFrom1900To1970 = 2208988800
	BUNDLE_TAG            = "#bundle"
)

// OscPacket is the interface for OscMessage and OscBundle.
type OscPacket interface {
	ToByteArray() (buffer []byte, err error)
}

// Represents a single OSC message. An OSC message consists of an OSC address
// pattern and zero or more arguments.
type OscMessage struct {
	Address   string
	Arguments []interface{}
}

// An OSC Bundle consists of the OSC-string "#bundle" followed by an OSC Time Tag,
// followed by zero or more OSC bundle/message elements. The OSC-timetag is a 64-bit fixed
// point time tag. See http://opensoundcontrol.org/spec-1_0 for more information.
type OscBundle struct {
	Timetag  OscTimetag
	Messages []*OscMessage
	Bundles  []*OscBundle
}

// An OSC client. It sends OSC messages and bundles to the given IP address
// and port.
type OscClient struct {
	ipaddress string
	port      int
}

// An OSC server. The server listens on Address and Port for incoming OSC packets
// and bundles.
type OscServer struct {
	Address     string         // Address to listen on
	Port        int            // Port to listen on
	ReadTimeout time.Duration  // Read Timeout
	dispatcher  *OscDispatcher // Dispatcher that dispatches OSC packets/messages
	running     bool
}

// OscTimetag represents an OSC Time Tag.
// An OSC Time Tag is defined as follows:
// Time tags are represented by a 64 bit fixed point number. The first 32 bits
// specify the number of seconds since midnight on January 1, 1900, and the
// last 32 bits specify fractional parts of a second to a precision of about
// 200 picoseconds. This is the representation used by Internet NTP timestamps.
type OscTimetag struct {
	timeTag  uint64 // The acutal time tag
	time     time.Time
	MinValue uint64 // Minimum value of an OSC Time Tag. Is always 1.
}

// Interface for an OSC message dispatcher. A dispatcher is responsible for
// dispatching received OSC messages.
type Dispatcher interface {
	Dispatch(packet OscPacket)
}

// OSC message handler interface. Every handler function for an OSC message must
// implement this interface.
type Handler interface {
	HandleMessage(msg *OscMessage)
}

// Type defintion for an OSC handler function
type HandlerFunc func(msg *OscMessage)

// HandleMessage calls themeself with the given OSC Message. Implements the
// Handler interface.
func (f HandlerFunc) HandleMessage(msg *OscMessage) {
	f(msg)
}

////
// OscDispatcher
////

// Dispatcher for OSC packets.
type OscDispatcher struct {
	handlers map[string]Handler
}

// NewOscDispatcher returns an OscDispatcher.
func NewOscDispatcher() (dispatcher *OscDispatcher) {
	return &OscDispatcher{handlers: make(map[string]Handler)}
}

// AddMsgHandler adds a new message handler for the given OSC address.
func (self *OscDispatcher) AddMsgHandler(address string, handler HandlerFunc) error {
	for _, chr := range "*?,[]{}# " {
		if strings.Contains(address, fmt.Sprintf("%c", chr)) {
			return errors.New("OSC Address string may not contain any characters in \"*?,[]{}# \n")
		}
	}

	if existsAddress(address, self.handlers) {
		return errors.New("OSC address exists already")
	}

	self.handlers[address] = handler

	return nil
}

// Dispatch dispatches OSC packets. Implements the Dispatcher interface.
func (self *OscDispatcher) Dispatch(packet OscPacket) {
	switch packet.(type) {
	default:
		return

	case *OscMessage:
		msg, _ := packet.(*OscMessage)
		for address, handler := range self.handlers {
			if msg.Match(address) {
				handler.HandleMessage(msg)
			}
		}

	case *OscBundle:
		bundle, _ := packet.(*OscBundle)
		timer := time.NewTimer(bundle.Timetag.ExpiresIn())

		go func() {
			<-timer.C
			for _, message := range bundle.Messages {
				for address, handler := range self.handlers {
					if message.Match(address) {
						handler.HandleMessage(message)
					}
				}
			}

			// Process bundles
			for _, b := range bundle.Bundles {
				self.Dispatch(b)
			}
		}()
	}
}

////
// OscMessage
////

// NewMessage returns a new OscMessage. The address parameter is the OSC address.
func NewOscMessage(address string) (msg *OscMessage) {
	return &OscMessage{Address: address}
}

// Append appends the given argument to the arguments list.
func (msg *OscMessage) Append(argument interface{}) {
	msg.Arguments = append(msg.Arguments, argument)
}

// Equals determines if the given OSC Message b is equal to the current OSC Message.
// It checks if the OSC address and the arguments are equal. Returns, true if the
// current object and b are equal.
func (msg *OscMessage) Equals(b *OscMessage) bool {
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

		case OscTimetag:
			if arg.(*OscTimetag).TimeTag() != b.Arguments[i].(*OscTimetag).TimeTag() {
				return false
			}
		}
	}

	return true
}

// Clear clears the OSC address and all arguments.
func (msg *OscMessage) Clear() {
	msg.Address = ""
	msg.ClearData()
}

// ClearData removes all arguments from the OSC Message.
func (msg *OscMessage) ClearData() {
	msg.Arguments = msg.Arguments[len(msg.Arguments):]
}

// Returns true, if the address of the OSC Message matches the given address.
// Case sensitive!
func (msg *OscMessage) Match(address string) bool {
	exp := getRegEx(msg.Address)

	if exp.MatchString(address) {
		return true
	}

	return false
}

// CountArguments returns the number of arguments.
func (msg *OscMessage) CountArguments() int {
	return len(msg.Arguments)
}

// ToByteBuffer serializes the OSC message to a byte buffer. The byte buffer
// is of the following format:
// 1. OSC Address Pattern
// 2. OSC Type Tag String
// 3. OSC Arguments
func (msg *OscMessage) ToByteArray() (buffer []byte, err error) {
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
			return nil, errors.New(fmt.Sprintf("OSC - unsupported type: %T", t))

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

		case OscTimetag:
			typetags = append(typetags, 't')

			timeTag := arg.(OscTimetag)
			payload.Write(timeTag.ToByteArray())
		}
	}

	// Write the type tag string to the data buffer
	_, err = writePaddedString(string(typetags), data)
	if err != nil {
		return nil, err
	}

	// Write the payload (OSC arguments) to the data buffer
	data.Write(payload.Bytes())

	return data.Bytes(), nil
}

////
// OscBundle
////

// NewOscBundle returns an OSC Bundle. Use this function to create a new OSC
// Bundle.
func NewOscBundle(time time.Time) (bundle *OscBundle) {
	return &OscBundle{Timetag: *NewOscTimetag(time)}
}

// Append appends an OSC packet (OSC bundle or message) to the bundle.
func (self *OscBundle) Append(pck OscPacket) (err error) {
	switch t := pck.(type) {
	default:
		return errors.New(fmt.Sprintf("Unsupported OSC packet type: only OscBundle and OscMessage are supported.", t))

	case *OscBundle:
		self.Bundles = append(self.Bundles, t)

	case *OscMessage:
		self.Messages = append(self.Messages, t)
	}

	return nil
}

// ToByteArray serializes the OSC bundle to a byte array with the following format:
// 1. Bundle string: '#bundle'
// 2. OSC timetag
// 3. Length of first OSC bundle element
// 4. First bundle element
// 5. Length of x OSC bundle element
// 6. x bundle element
func (self *OscBundle) ToByteArray() (buffer []byte, err error) {
	var data = new(bytes.Buffer)

	// Add the '#bundle' string
	_, err = writePaddedString("#bundle", data)
	if err != nil {
		return nil, err
	}

	// Add the timetag
	if _, err = data.Write(self.Timetag.ToByteArray()); err != nil {
		return nil, err
	}

	// Process all OSC Messages
	for _, m := range self.Messages {
		var msgLen int
		var msgBuf []byte

		msgBuf, err = m.ToByteArray()
		if err != nil {
			return nil, err
		}

		// Append the length of the OSC message
		msgLen = len(msgBuf)
		if err = binary.Write(data, binary.BigEndian, int32(msgLen)); err != nil {
			return nil, err
		}

		// Append the OSC message
		data.Write(msgBuf)
	}

	// Process all OSC Bundles
	for _, b := range self.Bundles {
		var bLen int
		var bBuf []byte

		bBuf, err = b.ToByteArray()
		if err != nil {
			return nil, err
		}

		// Write the size of the bundle
		bLen = len(bBuf)
		if err = binary.Write(data, binary.BigEndian, int32(bLen)); err != nil {
			return nil, err
		}

		// Append the bundle
		data.Write(bBuf)
	}

	return data.Bytes(), nil
}

////
// OscClient
////

// NewOscClient creates a new OSC client. The OscClient is used to send OSC
// messages and OSC bundles over an UDP network connection. The argument ip
// specifies the IP address and port defines the target port where the messages
// and bundles will be send to.
func NewOscClient(ip string, port int) (client *OscClient) {
	return &OscClient{ipaddress: ip, port: port}
}

// Ip returns the IP address.
func (client *OscClient) Ip() string {
	return client.ipaddress
}

// SetIp sets a new IP address.
func (client *OscClient) SetIp(ip string) {
	client.ipaddress = ip
}

// Port returns the port.
func (client *OscClient) Port() int {
	return client.port
}

// SetPort sets a new port.
func (client *OscClient) SetPort(port int) {
	client.port = port
}

// Send sends an OSC Bundle or an OSC Message.
func (client *OscClient) Send(packet OscPacket) (err error) {
	addr, _ := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", client.ipaddress, client.port))
	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		return err
	}

	data, err := packet.ToByteArray()
	if err != nil {
		conn.Close()
		return err
	}

	_, err = conn.Write(data)
	if err != nil {
		conn.Close()
		return err
	}

	conn.Close()

	return nil
}

////
// OscServer
////

// NewOscServer returns a new OscServer.
func NewOscServer(address string, port int) (server *OscServer) {
	return &OscServer{
		Address:     address,
		Port:        port,
		dispatcher:  NewOscDispatcher(),
		ReadTimeout: 0,
		running:     true}
}

// ListenAndServe retrieves incoming OSC packets.
// TODO: Add support for server running in a goroutine
func (self *OscServer) ListenAndServe() error {
	if self.dispatcher == nil {
		return errors.New("No dispatcher definied")
	}

	service := fmt.Sprintf("%s:%d", self.Address, self.Port)
	udpAddr, err := net.ResolveUDPAddr("udp", service)
	if err != nil {
		return err
	}

	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return err
	}

	// Set read timeout
	if self.ReadTimeout != 0 {
		conn.SetReadDeadline(time.Now().Add(self.ReadTimeout))
	}

	self.running = true
	for self.running {
		msg, err := self.readFromConnection(conn)
		if err == nil {
			go self.dispatcher.Dispatch(msg)
		}
	}

	// Close connection
	if err = conn.Close(); err != nil {
		return err
	}

	return nil
}

func (self *OscServer) Close() {
	self.running = false
}

func (self *OscServer) AddMsgHandler(address string, handler HandlerFunc) error {
	return self.dispatcher.AddMsgHandler(address, handler)
}

// readFromConnection retrieves OSC packets from the given io.Reader. If an OSC
// message is received an OSC Bundle will created and the message is appended to the
// bundle.
func (self *OscServer) readFromConnection(conn *net.UDPConn) (packet OscPacket, err error) {
	data := make([]byte, 65535)
	var n, start int
	n, _, err = conn.ReadFromUDP(data)
	packet, err = self.readPacket(bufio.NewReader(bytes.NewBuffer(data)), &start, n)

	return packet, nil
}

// receivePacket receives an OSC packet from the given reader.
func (self *OscServer) readPacket(reader *bufio.Reader, start *int, end int) (packet OscPacket, err error) {
	var buf []byte
	buf, err = reader.Peek(1)
	if err != nil {
		return nil, err
	}

	// An OSC Message starts with a '/'
	if buf[0] == '/' {
		packet, err = self.readMessage(reader, start)
		if err != nil {
			return nil, err
		}
	} else if buf[0] == '#' { // An OSC bundle starts with a '#'
		packet, err = self.readBundle(reader, start, end)
		if err != nil {
			return nil, err
		}
	}

	return packet, nil
}

// readBundle reads a OscBundle from reader.
func (self *OscServer) readBundle(reader *bufio.Reader, start *int, end int) (bundle *OscBundle, err error) {
	// Read the '#bundle' OSC string
	var startTag string
	var n int
	startTag, n, err = readPaddedString(reader)
	if err != nil {
		return nil, err
	}
	*start += n

	if startTag != BUNDLE_TAG {
		return nil, errors.New(fmt.Sprintf("Invalid bundle start tag: %s", startTag))
	}

	// Read the timetag
	var timeTag uint64
	if err := binary.Read(reader, binary.BigEndian, &timeTag); err != nil {
		return nil, err
	}
	*start += 8

	// Create a new bundle
	bundle = NewOscBundle(timetagToTime(timeTag))

	// Read until the end of the buffer
	for *start < end {
		// Read the size of the bundle element
		var length int32
		err = binary.Read(reader, binary.BigEndian, &length)
		*start += 4
		if err != nil {
			return nil, err
		}

		var packet OscPacket
		packet, err = self.readPacket(reader, start, end)
		if err != nil {
			return nil, err
		}
		bundle.Append(packet)
	}

	return bundle, nil
}

// readMessage reads one OSC Message from reader.
func (self *OscServer) readMessage(reader *bufio.Reader, start *int) (msg *OscMessage, err error) {
	// First, read the OSC address
	var n int
	address, n, err := readPaddedString(reader)
	if err != nil {
		return nil, err
	}
	*start += n

	// Create a new message
	msg = NewOscMessage(address)

	// Read all arguments
	if err = self.readArguments(msg, reader, start); err != nil {
		return nil, err
	}

	return msg, nil
}

// readArguments reads all arguments from the reader and adds it to the OSC message.
func (self *OscServer) readArguments(msg *OscMessage, reader *bufio.Reader, start *int) error {
	// Read the type tag string
	var n int
	typetags, n, err := readPaddedString(reader)
	if err != nil {
		return err
	}
	*start += n

	// If the typetag doesn't start with ',', it's not valid
	if typetags[0] != ',' {
		return errors.New("Unsupported type tag string")
	}

	// Remove ',' from the type tag
	typetags = typetags[1:]

	for _, c := range typetags {
		switch c {
		default:
			return errors.New(fmt.Sprintf("Unsupported type tag: %c", c))

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
			msg.Append(NewOscTimetagFromTimetag(tt))

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
// OscTimetag
////

// NewOscTimetag returns a new OSC timetag object.
func NewOscTimetag(timeStamp time.Time) (timetag *OscTimetag) {
	return &OscTimetag{
		time:     timeStamp,
		timeTag:  timeToTimetag(timeStamp),
		MinValue: uint64(1)}
}

// NewOscTimetagFromTimetag creates a new OscTimetag from the given time tag.
func NewOscTimetagFromTimetag(timetag uint64) (t *OscTimetag) {
	time := timetagToTime(timetag)
	return NewOscTimetag(time)
}

// Time returns the time.
func (self *OscTimetag) Time() time.Time {
	return self.time
}

// FractionalSecond returns the last 32 bits of the Osc Time Tag. Specifies the
// fractional part of a second.
func (self *OscTimetag) FractionalSecond() uint32 {
	return uint32(self.timeTag << 32)
}

// SecondsSinceEpoch returns the first 32 bits (the number of seconds since the
// midnight 1900) from the OSC timetag.
func (self *OscTimetag) SecondsSinceEpoch() uint32 {
	return uint32(self.timeTag >> 32)
}

// TimeTag returns the time tag value
func (self *OscTimetag) TimeTag() uint64 {
	return self.timeTag
}

// ToByteArray converts the OSC Time Tag to a byte array.
func (self *OscTimetag) ToByteArray() []byte {
	var data = new(bytes.Buffer)
	binary.Write(data, binary.BigEndian, self.timeTag)
	return data.Bytes()
}

// SetTime sets the value of the OSC Time Tag.
func (self *OscTimetag) SetTime(time time.Time) {
	self.time = time
	self.timeTag = timeToTimetag(time)
}

// ExpiresIn calculates the number of seconds until the current time is the
// same as the value of the timetag. It returns zero if the value of the
// timetag is in the past.
func (self *OscTimetag) ExpiresIn() time.Duration {
	if self.timeTag <= 1 {
		return 0
	}

	tt := timetagToTime(self.timeTag)
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
		n += padLen + 1
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
// Utility and various functions
////

// PrintOscMessages prints a OscMessage.
func PrintOscMessage(msg OscMessage) {
	// TODO: Implement PrintOscMessage
}

// existsAddress returns true if the address s is found in handlers. Otherwise, false.
func existsAddress(s string, handlers map[string]Handler) bool {
	for address, _ := range handlers {
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
