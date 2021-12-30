package main

import (
	_ "embed"
	"encoding/csv"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/jszwec/csvutil"
)

var (
	//go:embed "plotly-2.8.3.min.js"
	plotlyJS string

	//go:embed "index.html"
	indexHTML     string
	indexTemplate = template.Must(template.New("index").Parse(indexHTML))
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

type Table struct {
	Date   []time.Time
	Price  []float64
	Volume []int
}

func parseData(r io.Reader) (Table, error) {
	dec, err := csvutil.NewDecoder(csv.NewReader(r))
	if err != nil {
		return Table{}, err
	}
	dec.Register(unmarshalTime)

	var table Table
	for {
		var row Row
		err := dec.Decode(&row)

		if err == io.EOF {
			break
		}

		if err != nil {
			return Table{}, err
		}

		table.Date = append(table.Date, row.Date)
		table.Price = append(table.Price, row.Close)
		table.Volume = append(table.Volume, row.Volume)
	}

	return table, nil
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

func getStocks(symbol string, start, end time.Time) (Table, error) {
	u := buildURL(symbol, start, end)
	resp, err := http.Get(u)
	if err != nil {
		return Table{}, err
	}
	if resp.StatusCode != http.StatusOK {
		return Table{}, fmt.Errorf("%s", resp.Status)
	}
	defer resp.Body.Close()

	return parseData(resp.Body)
}

func jsHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, plotlyJS)
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	if err := indexTemplate.Execute(w, nil); err != nil {
		log.Printf("template: %s", err)
	}
}

func main() {
	http.HandleFunc("/static/plotly-2.8.3.min.js", jsHandler)
	http.HandleFunc("/", indexHandler)

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}

	/*
		symbol := "MSFT"
		start := time.Date(2021, time.January, 1, 0, 0, 0, 0, time.UTC)
		end := time.Date(2021, time.June, 31, 0, 0, 0, 0, time.UTC)
		rows, err := getStocks(symbol, start, end)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(rows)
	*/
}
