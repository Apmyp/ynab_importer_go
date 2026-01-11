package exchangerate

import (
	"encoding/json"
	"errors"
	"os"
	"sync"
	"time"
)

var ErrRateNotFound = errors.New("exchange rate not found")

type Rate struct {
	Date     time.Time
	Currency string
	Value    float64
}

type rateJSON struct {
	Date     string  `json:"date"`
	Currency string  `json:"currency"`
	Value    float64 `json:"value"`
}

type dataFile struct {
	Rates []rateJSON `json:"rates"`
}

type Store struct {
	filePath string
	mu       sync.RWMutex
}

func NewStore(filePath string) (*Store, error) {
	store := &Store{
		filePath: filePath,
	}

	if err := store.ensureFileExists(); err != nil {
		return nil, err
	}

	return store, nil
}

func (s *Store) ensureFileExists() error {
	if _, err := os.Stat(s.filePath); os.IsNotExist(err) {
		data := dataFile{Rates: []rateJSON{}}
		return s.writeFile(&data)
	}
	return nil
}

func (s *Store) readFile() (*dataFile, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	content, err := os.ReadFile(s.filePath)
	if err != nil {
		return nil, err
	}

	var data dataFile
	if err := json.Unmarshal(content, &data); err != nil {
		return nil, err
	}

	return &data, nil
}

func (s *Store) writeFile(data *dataFile) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	content, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.filePath, content, 0600)
}

func (s *Store) SaveRate(rate *Rate) error {
	data, err := s.readFile()
	if err != nil {
		return err
	}

	dateStr := rate.Date.Format("2006-01-02")

	found := false
	for i, r := range data.Rates {
		if r.Date == dateStr && r.Currency == rate.Currency {
			data.Rates[i].Value = rate.Value
			found = true
			break
		}
	}

	if !found {
		data.Rates = append(data.Rates, rateJSON{
			Date:     dateStr,
			Currency: rate.Currency,
			Value:    rate.Value,
		})
	}

	return s.writeFile(data)
}

func (s *Store) GetRate(date time.Time, currency string) (*Rate, error) {
	data, err := s.readFile()
	if err != nil {
		return nil, err
	}

	dateStr := date.Format("2006-01-02")

	for _, r := range data.Rates {
		if r.Date == dateStr && r.Currency == currency {
			parsedDate, err := time.Parse("2006-01-02", r.Date)
			if err != nil {
				return nil, err
			}
			return &Rate{
				Date:     parsedDate,
				Currency: r.Currency,
				Value:    r.Value,
			}, nil
		}
	}

	return nil, ErrRateNotFound
}

func (s *Store) Close() error {
	return nil
}
