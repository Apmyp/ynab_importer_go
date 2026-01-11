// Package message provides message types for SMS/iMessage data
package message

import (
	"fmt"
	"time"
)

// Message represents a parsed SMS message
type Message struct {
	Timestamp time.Time
	Sender    string
	Content   string
}

// String returns a string representation of the message
func (m *Message) String() string {
	return fmt.Sprintf("[%s] %s: %s", m.Timestamp.Format("2006-01-02 15:04:05"), m.Sender, m.Content)
}
