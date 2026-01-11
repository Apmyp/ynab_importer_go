package exchangerate

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type MockHTTPClient struct {
	response []byte
	err      error
}

func (m *MockHTTPClient) Get(url string) ([]byte, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.response, nil
}

func TestFetchRates_Success(t *testing.T) {
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

	mockClient := &MockHTTPClient{response: xmlResponse}
	fetcher := NewFetcherWithClient(mockClient)

	date := time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)
	rates, err := fetcher.FetchRates(date)
	if err != nil {
		t.Fatalf("FetchRates() error = %v", err)
	}

	if len(rates) != 2 {
		t.Fatalf("expected 2 rates, got %d", len(rates))
	}

	if rates[0].Currency != "USD" {
		t.Errorf("expected currency USD, got %s", rates[0].Currency)
	}
	if rates[0].Value != 18.1234 {
		t.Errorf("expected value 18.1234, got %f", rates[0].Value)
	}

	if rates[1].Currency != "EUR" {
		t.Errorf("expected currency EUR, got %s", rates[1].Currency)
	}
	if rates[1].Value != 19.7504 {
		t.Errorf("expected value 19.7504, got %f", rates[1].Value)
	}
}

func TestFetchRates_HTTPError(t *testing.T) {
	mockClient := &MockHTTPClient{err: errors.New("network error")}
	fetcher := NewFetcherWithClient(mockClient)

	date := time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)
	_, err := fetcher.FetchRates(date)
	if err == nil {
		t.Error("FetchRates() should return error on HTTP failure")
	}
}

func TestFetchRates_InvalidXML(t *testing.T) {
	mockClient := &MockHTTPClient{response: []byte("invalid xml")}
	fetcher := NewFetcherWithClient(mockClient)

	date := time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)
	_, err := fetcher.FetchRates(date)
	if err == nil {
		t.Error("FetchRates() should return error for invalid XML")
	}
	if !errors.Is(err, ErrInvalidXMLResponse) {
		t.Errorf("expected ErrInvalidXMLResponse, got %v", err)
	}
}

func TestFetchRates_EmptyResponse(t *testing.T) {
	mockClient := &MockHTTPClient{response: []byte("")}
	fetcher := NewFetcherWithClient(mockClient)

	date := time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)
	_, err := fetcher.FetchRates(date)
	if err == nil {
		t.Error("FetchRates() should return error for empty response")
	}
	if !errors.Is(err, ErrInvalidXMLResponse) {
		t.Errorf("expected ErrInvalidXMLResponse, got %v", err)
	}
}

func TestFetchRates_HTMLResponse(t *testing.T) {
	// Test when BNM returns HTML error page instead of XML
	mockClient := &MockHTTPClient{response: []byte(`<!DOCTYPE html><html><body>Error</body></html>`)}
	fetcher := NewFetcherWithClient(mockClient)

	date := time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)
	_, err := fetcher.FetchRates(date)
	if err == nil {
		t.Error("FetchRates() should return error for HTML response")
	}
	if !errors.Is(err, ErrInvalidXMLResponse) {
		t.Errorf("expected ErrInvalidXMLResponse, got %v", err)
	}
}

type URLCapturingClient struct {
	response    []byte
	capturedURL string
}

func (c *URLCapturingClient) Get(url string) ([]byte, error) {
	c.capturedURL = url
	return c.response, nil
}

func TestFetchRates_DateFormatting(t *testing.T) {
	mockClient := &URLCapturingClient{
		response: []byte(`<?xml version="1.0" encoding="utf-8"?><ValCurs Date="10.01.2026" Name="Official Exchange Rates"></ValCurs>`),
	}

	fetcher := NewFetcherWithClient(mockClient)

	date := time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)
	_, _ = fetcher.FetchRates(date)

	expectedURL := "https://www.bnm.md/en/official_exchange_rates?get_xml=1&date=10.01.2026"
	if mockClient.capturedURL != expectedURL {
		t.Errorf("expected URL %s, got %s", expectedURL, mockClient.capturedURL)
	}
}

func TestNewFetcher(t *testing.T) {
	fetcher := NewFetcher()
	if fetcher == nil {
		t.Error("NewFetcher() should return non-nil fetcher")
	}
	if fetcher.client == nil {
		t.Error("fetcher.client should not be nil")
	}
	if fetcher.baseURL == "" {
		t.Error("fetcher.baseURL should not be empty")
	}
}

func TestDefaultHTTPClient_Get_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test response"))
	}))
	defer server.Close()

	client := &DefaultHTTPClient{client: server.Client()}
	data, err := client.Get(server.URL)
	if err != nil {
		t.Errorf("Get() error = %v", err)
	}
	if string(data) != "test response" {
		t.Errorf("Get() data = %s, want test response", string(data))
	}
}

func TestDefaultHTTPClient_Get_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := &DefaultHTTPClient{client: server.Client()}
	_, err := client.Get(server.URL)
	if err == nil {
		t.Error("Get() should return error for non-200 status")
	}
}
