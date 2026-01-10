// Package bagoup handles running bagoup command and parsing its output
package bagoup

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
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

var messageLineRegex = regexp.MustCompile(`^\[(\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2})\] ([^:]+): (.*)$`)

// ParseMessageLine parses a single message line in bagoup format
func ParseMessageLine(line string) (*Message, error) {
	matches := messageLineRegex.FindStringSubmatch(line)
	if matches == nil {
		return nil, errors.New("invalid message format")
	}

	timestamp, err := time.Parse("2006-01-02 15:04:05", matches[1])
	if err != nil {
		return nil, fmt.Errorf("invalid timestamp: %w", err)
	}

	return &Message{
		Timestamp: timestamp,
		Sender:    matches[2],
		Content:   matches[3],
	}, nil
}

// ReadMessagesFromFile reads and parses all messages from a bagoup export file
func ReadMessagesFromFile(filePath string) ([]*Message, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var messages []*Message
	var currentMessage *Message

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		// Try to parse as a new message
		msg, err := ParseMessageLine(line)
		if err == nil {
			// Save previous message if exists
			if currentMessage != nil {
				messages = append(messages, currentMessage)
			}
			currentMessage = msg
		} else if currentMessage != nil {
			// Append to current message content
			currentMessage.Content += "\n" + line
		}
	}

	// Don't forget the last message
	if currentMessage != nil {
		messages = append(messages, currentMessage)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return messages, nil
}

// Runner executes the bagoup command
type Runner struct {
	dbPath    string
	outputDir string
	senders   []string
}

// NewRunner creates a new bagoup Runner
func NewRunner() *Runner {
	return &Runner{
		dbPath: "~/Library/Messages/chat.db",
	}
}

// WithDBPath sets the database path
func (r *Runner) WithDBPath(path string) *Runner {
	r.dbPath = path
	return r
}

// WithOutputDir sets the output directory
func (r *Runner) WithOutputDir(dir string) *Runner {
	r.outputDir = dir
	return r
}

// WithSenders sets the senders to filter
func (r *Runner) WithSenders(senders []string) *Runner {
	r.senders = senders
	return r
}

// CheckDependencies verifies that bagoup is available
func (r *Runner) CheckDependencies() error {
	_, err := exec.LookPath("bagoup")
	if err != nil {
		return errors.New("bagoup command not found in PATH")
	}
	return nil
}

// Run executes bagoup and returns the output directory
func (r *Runner) Run() (string, error) {
	if err := r.CheckDependencies(); err != nil {
		return "", err
	}

	if r.outputDir == "" {
		r.outputDir = fmt.Sprintf("messages_%d", time.Now().UnixNano())
	}

	args := []string{
		"--separate-chats",
		"-i", r.dbPath,
		"-o", r.outputDir,
	}

	for _, sender := range r.senders {
		args = append(args, "-e", sender)
	}

	cmd := exec.Command("bagoup", args...)

	// Set ulimit before running (needed for bagoup)
	cmd.Env = append(os.Environ(), "ULIMIT_N=2048")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("bagoup failed: %s: %w", string(output), err)
	}

	return r.outputDir, nil
}

// Cleanup removes the output directory
func (r *Runner) Cleanup() error {
	if r.outputDir != "" {
		return os.RemoveAll(r.outputDir)
	}
	return nil
}

// ReadAllMessages reads all messages from the bagoup output directory
func (r *Runner) ReadAllMessages() ([]*Message, error) {
	var allMessages []*Message

	for _, sender := range r.senders {
		senderDir := fmt.Sprintf("%s/%s", r.outputDir, sender)
		files, err := os.ReadDir(senderDir)
		if err != nil {
			// Sender directory might not exist if no messages
			continue
		}

		for _, file := range files {
			if file.IsDir() || !strings.HasSuffix(file.Name(), ".txt") {
				continue
			}

			filePath := fmt.Sprintf("%s/%s", senderDir, file.Name())
			messages, err := ReadMessagesFromFile(filePath)
			if err != nil {
				return nil, fmt.Errorf("reading %s: %w", filePath, err)
			}

			allMessages = append(allMessages, messages...)
		}
	}

	return allMessages, nil
}
