package osc

// TODO: Implement all test cases

import "testing"

func TestAddMsgHandlerWithInvalidAddress(t *testing.T) {
	server := NewOscServer("localhost", 6677)
	err := server.AddMsgHandler("/address*/test", func(msg *OscMessage) {})
	if err == nil {
		t.Error("Expected error with '/address*/test'")
	}
}

func TestAddMsgHandler(t *testing.T) {
	server := NewOscServer("localhost", 6677)
	err := server.AddMsgHandler("/address/test", func(msg *OscMessage) {})
	if err != nil {
		t.Error("Expected that OSC address '/address/test' is valid")
	}
}
