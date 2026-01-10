package exchangerate

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestConverter_GetOrFetchRate_Cached(t *testing.T) {
	dir := t.TempDir()
	storePath := filepath.Join(dir, "data.json")
	store, err := NewStore(storePath)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()

	date := time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)
	cachedRate := &Rate{
		Date:     date,
		Currency: "USD",
		Value:    18.5,
	}
	store.SaveRate(cachedRate)

	mockFetcher := &MockHTTPClient{}
	fetcher := NewFetcherWithClient(mockFetcher)

	converter := NewConverter(store, fetcher, "MDL")

	rate, err := converter.GetOrFetchRate(date, "USD")
	if err != nil {
		t.Fatalf("GetOrFetchRate() error = %v", err)
	}

	if rate != 18.5 {
		t.Errorf("expected rate 18.5, got %f", rate)
	}

	if mockFetcher.response != nil {
		t.Error("GetOrFetchRate() should not call API when rate is cached")
	}
}

func TestConverter_GetOrFetchRate_FetchMissing(t *testing.T) {
	dir := t.TempDir()
	storePath := filepath.Join(dir, "data.json")
	store, err := NewStore(storePath)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()

	xmlResponse := []byte(`<?xml version="1.0" encoding="utf-8"?>
<ValCurs Date="10.01.2026" Name="Official Exchange Rates">
  <Valute ID="1">
    <NumCode>840</NumCode>
    <CharCode>USD</CharCode>
    <Nominal>1</Nominal>
    <Name>US Dollar</Name>
    <Value>18.1234</Value>
  </Valute>
</ValCurs>`)

	mockFetcher := &MockHTTPClient{response: xmlResponse}
	fetcher := NewFetcherWithClient(mockFetcher)

	converter := NewConverter(store, fetcher, "MDL")

	date := time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)
	rate, err := converter.GetOrFetchRate(date, "USD")
	if err != nil {
		t.Fatalf("GetOrFetchRate() error = %v", err)
	}

	if rate != 18.1234 {
		t.Errorf("expected rate 18.1234, got %f", rate)
	}

	retrieved, err := store.GetRate(date, "USD")
	if err != nil {
		t.Fatalf("rate should be saved to store")
	}
	if retrieved.Value != 18.1234 {
		t.Errorf("saved rate should be 18.1234, got %f", retrieved.Value)
	}
}

func TestConverter_GetOrFetchRate_SameCurrency(t *testing.T) {
	dir := t.TempDir()
	storePath := filepath.Join(dir, "data.json")
	store, err := NewStore(storePath)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()

	mockFetcher := &MockHTTPClient{}
	fetcher := NewFetcherWithClient(mockFetcher)

	converter := NewConverter(store, fetcher, "MDL")

	date := time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)
	rate, err := converter.GetOrFetchRate(date, "MDL")
	if err != nil {
		t.Fatalf("GetOrFetchRate() error = %v", err)
	}

	if rate != 1.0 {
		t.Errorf("expected rate 1.0 for same currency, got %f", rate)
	}
}

func TestConverter_GetOrFetchRate_NilStore(t *testing.T) {
	xmlResponse := []byte(`<?xml version="1.0" encoding="utf-8"?>
<ValCurs Date="10.01.2026" Name="Official Exchange Rates">
  <Valute ID="1">
    <NumCode>840</NumCode>
    <CharCode>USD</CharCode>
    <Nominal>1</Nominal>
    <Name>US Dollar</Name>
    <Value>18.1234</Value>
  </Valute>
</ValCurs>`)

	mockFetcher := &MockHTTPClient{response: xmlResponse}
	fetcher := NewFetcherWithClient(mockFetcher)

	converter := NewConverter(nil, fetcher, "MDL")

	date := time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)
	rate, err := converter.GetOrFetchRate(date, "USD")
	if err != nil {
		t.Fatalf("GetOrFetchRate() error = %v", err)
	}

	if rate != 18.1234 {
		t.Errorf("expected rate 18.1234, got %f", rate)
	}
}

func TestConverter_GetOrFetchRate_NilStore_CurrencyNotFound(t *testing.T) {
	xmlResponse := []byte(`<?xml version="1.0" encoding="utf-8"?>
<ValCurs Date="10.01.2026" Name="Official Exchange Rates">
  <Valute ID="1">
    <NumCode>978</NumCode>
    <CharCode>EUR</CharCode>
    <Nominal>1</Nominal>
    <Name>Euro</Name>
    <Value>19.7504</Value>
  </Valute>
</ValCurs>`)

	mockFetcher := &MockHTTPClient{response: xmlResponse}
	fetcher := NewFetcherWithClient(mockFetcher)

	converter := NewConverter(nil, fetcher, "MDL")

	date := time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)
	_, err := converter.GetOrFetchRate(date, "USD")
	if err == nil {
		t.Error("GetOrFetchRate() should return error when currency not found")
	}
}

func TestConverter_GetOrFetchRate_FetchError(t *testing.T) {
	dir := t.TempDir()
	storePath := filepath.Join(dir, "data.json")
	store, err := NewStore(storePath)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()

	mockFetcher := &MockHTTPClient{err: errors.New("network error")}
	fetcher := NewFetcherWithClient(mockFetcher)

	converter := NewConverter(store, fetcher, "MDL")

	date := time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)
	_, err = converter.GetOrFetchRate(date, "USD")
	if err == nil {
		t.Error("GetOrFetchRate() should return error when fetch fails")
	}
}

func TestConverter_GetOrFetchRate_StoreError(t *testing.T) {
	dir := t.TempDir()
	storePath := filepath.Join(dir, "data.json")

	// Create store with invalid JSON
	os.WriteFile(storePath, []byte("bad json"), 0644)

	store, _ := NewStore(storePath)
	mockFetcher := &MockHTTPClient{}
	fetcher := NewFetcherWithClient(mockFetcher)

	converter := NewConverter(store, fetcher, "MDL")

	date := time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)
	_, err := converter.GetOrFetchRate(date, "USD")
	if err == nil {
		t.Error("GetOrFetchRate() should return error when store has corrupt data")
	}
}

func TestConverter_GetOrFetchRate_NilStoreWithError(t *testing.T) {
	mockFetcher := &MockHTTPClient{err: errors.New("network failure")}
	fetcher := NewFetcherWithClient(mockFetcher)

	converter := NewConverter(nil, fetcher, "MDL")

	date := time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)
	_, err := converter.GetOrFetchRate(date, "USD")
	if err == nil {
		t.Error("GetOrFetchRate() should return error when fetch fails with nil store")
	}
}

func TestConverter_GetOrFetchRate_CurrencyNotFoundInFetch(t *testing.T) {
	dir := t.TempDir()
	storePath := filepath.Join(dir, "data.json")
	store, err := NewStore(storePath)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()

	// API returns only EUR, not USD
	xmlResponse := []byte(`<?xml version="1.0" encoding="utf-8"?>
<ValCurs Date="10.01.2026" Name="Official Exchange Rates">
  <Valute ID="2">
    <NumCode>978</NumCode>
    <CharCode>EUR</CharCode>
    <Nominal>1</Nominal>
    <Name>Euro</Name>
    <Value>19.7504</Value>
  </Valute>
</ValCurs>`)

	mockFetcher := &MockHTTPClient{response: xmlResponse}
	fetcher := NewFetcherWithClient(mockFetcher)

	converter := NewConverter(store, fetcher, "MDL")

	date := time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)

	// Request USD which is not in the response
	_, err = converter.GetOrFetchRate(date, "USD")
	if err == nil {
		t.Error("GetOrFetchRate() should return error when currency not found in API response")
	}
	if err.Error() != "currency not found in exchange rates" {
		t.Errorf("expected error 'currency not found in exchange rates', got %v", err)
	}
}

func TestConverter_GetOrFetchRate_SavesOnlyRequestedCurrency(t *testing.T) {
	dir := t.TempDir()
	storePath := filepath.Join(dir, "data.json")
	store, err := NewStore(storePath)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()

	// API returns both USD and EUR
	xmlResponse := []byte(`<?xml version="1.0" encoding="utf-8"?>
<ValCurs Date="10.01.2026" Name="Official Exchange Rates">
  <Valute ID="1">
    <NumCode>840</NumCode>
    <CharCode>USD</CharCode>
    <Nominal>1</Nominal>
    <Name>US Dollar</Name>
    <Value>18.1234</Value>
  </Valute>
  <Valute ID="2">
    <NumCode>978</NumCode>
    <CharCode>EUR</CharCode>
    <Nominal>1</Nominal>
    <Name>Euro</Name>
    <Value>19.7504</Value>
  </Valute>
</ValCurs>`)

	mockFetcher := &MockHTTPClient{response: xmlResponse}
	fetcher := NewFetcherWithClient(mockFetcher)

	converter := NewConverter(store, fetcher, "MDL")

	date := time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)

	// Request only USD
	rate, err := converter.GetOrFetchRate(date, "USD")
	if err != nil {
		t.Fatalf("GetOrFetchRate() error = %v", err)
	}
	if rate != 18.1234 {
		t.Errorf("expected rate 18.1234, got %f", rate)
	}

	// Verify USD was saved
	usdRate, err := store.GetRate(date, "USD")
	if err != nil {
		t.Fatalf("USD should be saved to store")
	}
	if usdRate.Value != 18.1234 {
		t.Errorf("saved USD rate should be 18.1234, got %f", usdRate.Value)
	}

	// Verify EUR was NOT saved (only requested currency should be saved)
	_, err = store.GetRate(date, "EUR")
	if err != ErrRateNotFound {
		t.Error("EUR should NOT be saved to store (only requested currency should be saved)")
	}
}
