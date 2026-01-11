package message

import (
	"testing"
	"time"
)

func TestMessage_String(t *testing.T) {
	msg := &Message{
		Timestamp: time.Date(2023, 5, 3, 16, 21, 47, 0, time.UTC),
		Sender:    "102",
		Content:   "Test content",
	}

	str := msg.String()
	expected := "[2023-05-03 16:21:47] 102: Test content"
	if str != expected {
		t.Errorf("Message.String() = %q, want %q", str, expected)
	}
}
