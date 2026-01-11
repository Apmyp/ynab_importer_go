package message

import (
	"fmt"
	"time"
)

type Message struct {
	Timestamp time.Time
	Sender    string
	Content   string
}

func (m *Message) String() string {
	return fmt.Sprintf("[%s] %s: %s", m.Timestamp.Format("2006-01-02 15:04:05"), m.Sender, m.Content)
}
