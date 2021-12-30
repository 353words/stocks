package main

import (
	_ "embed"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/jszwec/csvutil"
)

var (
	//go:embed "index.html"
	indexHTML []byte
	//go:embed "plotly-2.8.3.min.js"
	plotlyJS []byte
	//go:embed "chart.js"
	chartJS []byte
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

func staticHandler(w http.ResponseWriter, r *http.Request) {
	var data []byte
	switch r.URL.Path {
	case "/":
		data = indexHTML
	case "/js/plotly-2.8.3.min.js":
		data = plotlyJS
	case "/js/chart.js":
		data = chartJS
	}

	if data == nil {
		log.Printf("%q not found", r.URL.Path)
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.Write(data)
}

func dataHandler(w http.ResponseWriter, r *http.Request) {
	symbol := r.URL.Query().Get("symbol")
	if symbol == "" {
		http.Error(w, "empty symbol", http.StatusBadRequest)
		return
	}
	log.Printf("data: %q", symbol)
	start := time.Date(2021, time.January, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2021, time.June, 31, 0, 0, 0, 0, time.UTC)
	table, err := getStocks(symbol, start, end)
	if err != nil {
		log.Printf("get %q: %s", symbol, err)
		http.Error(w, "can't fetch data", http.StatusInternalServerError)
		return
	}

	if err := tableJSON(symbol, table, w); err != nil {
		log.Printf("table: %s", err)
	}
}

func tableJSON(symbol string, table Table, w io.Writer) error {
	var reply struct {
		Data [2]struct {
			X    interface{} `json:"x"`
			Y    interface{} `json:"y"`
			Name string      `json:"name"`
			Mode string      `json:"mode"`
		} `json:"data"`
		Layout struct {
			Title string `json:"title"`
			Grid  struct {
				Rows    int `json:"rows"`
				Columns int `json:"columns"`
			} `json:"grid"`
		} `json:"layout"`
	}

	reply.Layout.Title = symbol
	reply.Layout.Grid.Rows = 2
	reply.Layout.Grid.Columns = 1
	reply.Data[0].X = table.Date
	reply.Data[0].Y = table.Price
	reply.Data[0].Name = "Price"
	reply.Data[0].Mode = "line"
	reply.Data[1].X = table.Date
	reply.Data[1].Y = table.Volume
	reply.Data[1].Name = "Volume"
	reply.Data[1].Mode = "bar"

	return json.NewEncoder(w).Encode(reply)
}

func main() {
	http.HandleFunc("/", staticHandler)
	http.HandleFunc("/data", dataHandler)

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
