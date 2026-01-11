package exchangerate

import (
	"errors"
	"time"
)

type Converter struct {
	store           *Store
	fetcher         *Fetcher
	defaultCurrency string
}

func NewConverter(store *Store, fetcher *Fetcher, defaultCurrency string) *Converter {
	return &Converter{
		store:           store,
		fetcher:         fetcher,
		defaultCurrency: defaultCurrency,
	}
}

func (c *Converter) GetOrFetchRate(date time.Time, currency string) (float64, error) {
	if currency == c.defaultCurrency {
		return 1.0, nil
	}

	// Try to get from store if available
	if c.store != nil {
		rate, err := c.store.GetRate(date, currency)
		if err == nil {
			return rate.Value, nil
		}
		if !errors.Is(err, ErrRateNotFound) {
			return 0, err
		}
	}

	// Fetch from API
	rates, err := c.fetcher.FetchRates(date)
	if err != nil {
		return 0, err
	}

	// Find the requested currency
	for _, r := range rates {
		if r.Currency == currency {
			// Save to store if available
			if c.store != nil {
				c.store.SaveRate(r)
			}
			return r.Value, nil
		}
	}

	return 0, errors.New("currency not found in exchange rates")
}
