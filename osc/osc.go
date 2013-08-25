/*
 * goosc provides a package for sending and receiving OpenSoundControl messages.
 * The package is implemented in pure Go.
 *
 * Open Sound Control (OSC) is an open, transport-independent, message-based
 * protocol developed for communication among computers, sound synthesizers,
 * and other multimedia devices.
 *
 * This OSC implementation uses the UDP/IP protocol for sending and receiving
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
 * OSC Messages: An OSC message consists of an OSC Address Pattern followed
 * by an OSC Type Tag String followed by zero or more OSC Arguments.
 *
 * OSC Bundles: An OSC Bundle consists of the OSC-string "#bundle" followed
 * by an OSC Time Tag, followed by zero or more OSC Bundle Elements.
 *
 * An OSC Bundle Element consists of its size and its contents. The size is
 * an int32 representing the number of 8-bit bytes in the contents, and will
 * always be a multiple of 4. The contents are either an OSC Message or an
 * OSC Bundle.
 *
 * The following types are supported: 'i' (Int32), 'f' (Float32), 's' (string),
 * 'b' (blob / binary data), 'h' (Int64), 't' (OSC timetag), 'd' (Double),
 * 'r' (RGBA color), 'T' (True), 'F' (False), 'N' (Nil).
 *
 * Supported OSC address patterns: TODO
 *
 *
 * Author: Sebastian Ruml <sebastian.ruml@gmail.com>
 * Created: 2013.08.19
 *
 */

package osc

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"time"
)

const (
	// The time tag value consisting of 63 zero bits followed by a one in the
	// least signifigant bit is a special case meaning "immediately."
	timeTagImmediate      = uint64(1)
	secondsFrom1900To1970 = 2208988800
)

// A single OSC message.
type OscMessage struct {
	Address   string
	arguments []interface{}
}

// An OSC bundle.
type OscBundle struct {
	Timetag  OscTimetag
	messages []*OscMessage
	bundles  []*OscBundle
}

// An OSC client. It sends OSC messages and bundles to a given endpoint
// (IP address and Port).
type OscClient struct {
	endpoint Endpoint
}

// An OSC server. It listens on Port for incoming OSC messages and bundles.
type OscServer struct {
	Port int
}

// OSC IP endpoint, consisting of an IP address and a port.
type Endpoint struct {
	address string
	port    int
}

// OscTimetag represents an OSC timetag.
// An OSC timetage is defined as follows:
// Time tags are represented by a 64 bit fixed point number. The first 32 bits
// specify the number of seconds since midnight on January 1, 1900, and the
// last 32 bits specify fractional parts of a second to a precision of about
// 200 picoseconds. This is the representation used by Internet NTP timestamps.
type OscTimetag struct {
	timeTag  uint64
	time     time.Time
	MinValue uint64
}

// NewMessage returns a new OscMessage. The argument address is the OSC address.
func NewOscMessage(address string) (msg *OscMessage) {
	return &OscMessage{Address: address}
}

// Append adds the given argument to the end of the arguments list.
func (msg *OscMessage) Append(argument interface{}) (err error) {
	if argument == nil {
		return err
	}

	msg.Arguments = append(msg.Arguments, argument)

	return nil
}

// Equals determines if the given OSC Message b is equal to the current OSC Message.
// It checks if the OSC address and the arguments are equal. Returns, true if the
// current object and b are equal.
func (msg *OscMessage) Equals(b *OscMessage) bool {
	// TODO
	return true
}

// Clear clears the OSC address and clear any arguments appended so far.
func (msg *OscMessage) Clear() {
	msg.Address = ""
	msg.ClearData()
}

// ClearData clears all data from the OscMessage. It removes all arguments from
// the
func (msg *OscMessage) ClearData() {
	msg.arguments = msg.arguments[len(msg.arguments):]
}

// Returns true, if the Address of the OscElement matches the given address.
// Case sensitive!
func (msg *OscMessage) Match(address string) bool {
	// TODO

	return true
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

	var payload = new(bytes.Buffer)
	for _, arg := range msg.Arguments {
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

			if err = binary.Write(payload, binary.BigEndian, arg); err != nil {
				return nil, err
			}

		case float32:
			typetags = append(typetags, 'f')

			if err = binary.Write(payload, binary.BigEndian, arg); err != nil {
				return nil, err
			}

		case string:
			typetags = append(typetags, 's')

			if _, err = writePaddedString(arg.(string), payload); err != nil {
				return nil, err
			}

		case []byte:
			typetags = append(typetags, 'b')

			if _, err = writeBlob(arg.([]byte), payload); err != nil {
				return nil, err
			}

		case int64:
			typetags = append(typetags, 'h')

			if err = binary.Write(payload, binary.BigEndian, arg); err != nil {
				return nil, err
			}

		case float64:
			typetags = append(typetags, 'd')

			if err = binary.Write(payload, binary.BigEndian, arg); err != nil {
				return nil, err
			}

		case OscTimetag:
			typetags = append(typetags, 't')

			timeTag := arg.(OscTimetag)
			payload.Write(timeTag.ToByteArray())
		}
	}

	// Write the typetags string
	_, err = writePaddedString(string(typetags), data)
	if err != nil {
		return nil, err
	}

	// Write the payload (OSC arguments) to the data buffer
	data.Write(payload.Bytes())

	return data.Bytes(), nil
}

// FromByteBuffer deserializes the OSC message from the given byte buffer.
func (msg *OscMessage) FromByteArray(buffer bytes.Buffer) {

}

// NewOscBundle returns an OSC Bundle.
func NewOscBundle(timetag time.Time) (bundle *OscBundle) {
	return &OscBundle{timetag: NewOscTimetag(timetag)}
}

// TODO
func (self *OscBundle) AppendMessage(msg *OscMessage) {
	self.messages = append(self.messages, msg)
}

// TODO
func (self *OscBundle) AppendBundle(bundle *OscBundle) {
	self.bundles = append(self.bundles, bundle)
}

// NewOscClient creates a new OSC client. The OscClient is used to send OSC
// messages and OSC bundles over an UDP network connection. The argument ip
// specifies the IP address and port defines the target port where the messages
// and bundles will be send to.
func NewOscClient(ip string, port int) (client *OscClient) {
	return &OscClient{endpoint: Endpoint{address: ip, port: port}}
}

// SendMessage sends an OscMessage.
func (client *OscClient) SendMessage(message *OscMessage) (err error) {
	addr, _ := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", client.endpoint.address, client.endpoint.port))
	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		return err
	}

	data, err := message.toByteArray()
	if err != nil {
		return errors.New("Can't convert OSC message to byte array")
	}

	// Write the data to the connection
	_, err = conn.Write(data)
	if err != nil {
		return err
	}

	return nil
}

// SendBundle sends an OSC Bundle.
func (client *OscClient) SendBundle(bundle OscBundle) (err error) {
	addr, _ := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", client.endpoint.address, client.endpoint.port))
	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		return err
	}

	return nil
}

// NewOscTimetag returns a new OSC timetag object.
func NewOscTimetag(timeStamp time.Time) (timetag *OscTimetag) {
	return &OscTimetag{
		time:     timeStamp,
		timeTag:  timeToTimetag(timeStamp),
		MinValue: uint64(1)}
}

// FractionalSecond returns the last 32 bits of the Osc Time Tag. Specifies the
// fractional part of a second.
func (self *OscTimetag) FractionalSecond() uint32 {
	return uint32(self.timeTag << 32)
}

// SecondsSinceEpoch returns the first 32 bits of the Osc Time Tag. Specifies
// the number of seconds since the epoch.
func (self *OscTimetag) SecondsSinceEpoch() uint32 {
	return uint32(self.timeTag >> 32)
}

// TimeTag returns the OSC time tag value
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

// readBlob reads an OSC Blob from the blob byte array.
func readBlob(data []byte) (blob []byte, err error) {
	blob = new(bytes.Buffer)

	return blob.Bytes(), nil
}

// writeBlob writes the data byte array as an OSC blob into buff. If the length of
// data isn't 32-bit aligned, the padding bytes will be added.
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

// writePaddedString writes the argument str with padding bytes to the argument
// buff. Returns, the number of written bytes.
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
