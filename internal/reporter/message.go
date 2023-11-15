package reporter

import (
	"fmt"
	"time"
)

// Messages types
const (
	OKMessage = iota
	WarnMessage
	ErrorMessage
)

type (
	MessageType uint8
	Message     struct {
		time    time.Time
		message string
		msgType MessageType
	}
)

// NewMessage would initialises Message and would return it
func NewMessage(msg string, msgType MessageType) *Message {
	return &Message{
		time:    time.Now(),
		message: msg,
		msgType: msgType,
	}
}

// MsgType would return type of this message
func (m *Message) MsgType() MessageType {
	return m.msgType
}

func (m *Message) String() string {
	return fmt.Sprintf("%s: %s", m.time.Format("2006-01-02 15:04:05"), m.message)
}
