package exchangerate

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var ErrInvalidXMLResponse = fmt.Errorf("BNM API returned invalid XML response")

type HTTPClient interface {
	Get(url string) ([]byte, error)
}

type DefaultHTTPClient struct {
	client *http.Client
}

func (c *DefaultHTTPClient) Get(url string) ([]byte, error) {
	resp, err := c.client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP request failed with status %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

type Fetcher struct {
	client  HTTPClient
	baseURL string
}

func NewFetcher() *Fetcher {
	return &Fetcher{
		client: &DefaultHTTPClient{
			client: &http.Client{
				Timeout: 10 * time.Second,
			},
		},
		baseURL: "https://www.bnm.md/en/official_exchange_rates",
	}
}

func NewFetcherWithClient(client HTTPClient) *Fetcher {
	return &Fetcher{
		client:  client,
		baseURL: "https://www.bnm.md/en/official_exchange_rates",
	}
}

type ValCurs struct {
	XMLName xml.Name `xml:"ValCurs"`
	Date    string   `xml:"Date,attr"`
	Valutes []Valute `xml:"Valute"`
}

type Valute struct {
	ID       string `xml:"ID,attr"`
	NumCode  string `xml:"NumCode"`
	CharCode string `xml:"CharCode"`
	Nominal  int    `xml:"Nominal"`
	Name     string `xml:"Name"`
	Value    string `xml:"Value"`
}

func (f *Fetcher) FetchRates(date time.Time) ([]*Rate, error) {
	dateStr := date.Format("02.01.2006")
	url := fmt.Sprintf("%s?get_xml=1&date=%s", f.baseURL, dateStr)

	data, err := f.client.Get(url)
	if err != nil {
		return nil, err
	}

	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || (trimmed[0] != '<') {
		return nil, ErrInvalidXMLResponse
	}

	var valCurs ValCurs
	if err := xml.Unmarshal(data, &valCurs); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidXMLResponse, err)
	}

	if valCurs.XMLName.Local != "ValCurs" {
		return nil, fmt.Errorf("%w: unexpected root element", ErrInvalidXMLResponse)
	}

	var rates []*Rate
	for _, valute := range valCurs.Valutes {
		value, err := parseValue(valute.Value)
		if err != nil {
			continue
		}

		if valute.Nominal > 1 {
			value = value / float64(valute.Nominal)
		}

		rates = append(rates, &Rate{
			Date:     date.UTC().Truncate(24 * time.Hour),
			Currency: valute.CharCode,
			Value:    value,
		})
	}

	return rates, nil
}

func parseValue(value string) (float64, error) {
	normalized := strings.Replace(value, ",", ".", -1)
	return strconv.ParseFloat(normalized, 64)
}
