package chatdb

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/apmyp/ynab_importer_go/message"
	_ "modernc.org/sqlite"
)

// Apple's absolute time reference: 2001-01-01 00:00:00 UTC
const appleEpoch = 978307200

type Reader struct {
	db      *sql.DB
	senders []string
}

func NewReader(dbPath string, senders []string) (*Reader, error) {
	db, err := sql.Open("sqlite", dbPath+"?mode=ro")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	return &Reader{
		db:      db,
		senders: senders,
	}, nil
}

func (r *Reader) Close() error {
	if r.db != nil {
		return r.db.Close()
	}
	return nil
}

func (r *Reader) FetchMessages() ([]*message.Message, error) {
	if len(r.senders) == 0 {
		return []*message.Message{}, nil
	}

	query := `
		SELECT
			m.ROWID,
			h.id as sender,
			m.text,
			m.date,
			m.is_from_me
		FROM message m
		JOIN handle h ON m.handle_id = h.ROWID
		WHERE m.is_from_me = 0
		AND h.id IN (` + buildPlaceholders(len(r.senders)) + `)
		ORDER BY m.date ASC
	`

	args := make([]interface{}, len(r.senders))
	for i, sender := range r.senders {
		args[i] = sender
	}

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query messages: %w", err)
	}
	defer rows.Close()

	var messages []*message.Message
	for rows.Next() {
		var (
			rowID    int64
			sender   string
			text     sql.NullString
			date     int64
			isFromMe int
		)

		if err := rows.Scan(&rowID, &sender, &text, &date, &isFromMe); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		if !text.Valid || text.String == "" {
			continue
		}

		timestamp := appleTimeToUnix(date)

		messages = append(messages, &message.Message{
			Timestamp: timestamp,
			Sender:    sender,
			Content:   text.String,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return messages, nil
}

func appleTimeToUnix(appleTimeNano int64) time.Time {
	// Messages database stores time as nanoseconds since 2001-01-01 (Apple's epoch)
	appleSeconds := appleTimeNano / 1000000000
	unixTimestamp := appleSeconds + appleEpoch
	return time.Unix(unixTimestamp, 0).UTC()
}

func buildPlaceholders(count int) string {
	if count == 0 {
		return ""
	}

	result := "?"
	for i := 1; i < count; i++ {
		result += ",?"
	}
	return result
}
