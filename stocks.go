package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/jszwec/csvutil"
)

type Row struct {
	Date   time.Time
	Open   float64
	High   float64
	Low    float64
	Close  float64
	Volume int
}

// unmarshalTime unmarshal data in CSV to time
func unmarshalTime(data []byte, t *time.Time) error {
	var err error
	*t, err = time.Parse("2006-01-02", string(data))
	return err
}

func parseData(r io.Reader) ([]Row, error) {
	dec, err := csvutil.NewDecoder(csv.NewReader(r))
	if err != nil {
		return nil, err
	}
	dec.Register(unmarshalTime)

	var rows []Row
	for {
		var row Row
		err := dec.Decode(&row)

		if err == io.EOF {
			break
		}

		if err != nil {
			return nil, err
		}

		rows = append(rows, row)
	}

	return rows, nil
}

func buildURL(symbol string, start, end time.Time) string {
	u := fmt.Sprintf("https://query1.finance.yahoo.com/v7/finance/download/%s", url.PathEscape(symbol))
	v := url.Values{
		"period1":  {fmt.Sprintf("%d", start.Unix())},
		"period2":  {fmt.Sprintf("%d", end.Unix())},
		"interval": {"1d"},
		"events":   {"history"},
	}

	return fmt.Sprintf("%s?%s", u, v.Encode())
}

func getStocks(symbol string, start, end time.Time) ([]Row, error) {
	u := buildURL(symbol, start, end)
	resp, err := http.Get(u)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s", resp.Status)
	}
	defer resp.Body.Close()

	return parseData(resp.Body)
}

func main() {
	symbol := "MSFT"
	start := time.Date(2021, time.January, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2021, time.June, 31, 0, 0, 0, 0, time.UTC)
	rows, err := getStocks(symbol, start, end)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(rows)
}
