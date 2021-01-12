package nullitics

import (
	"bytes"
	_ "embed"
	"html/template"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var Now = time.Now

var dailyLog = "log.csv"
var historyLog = "stats.csv"

var (
	MaxPathLength    = 200
	MaxRefLength     = 64
	MaxCountryLength = 2
)

type Collector struct {
	sync.Mutex
	dir      string
	location *time.Location
	salt     string
	appender *Appender
	history  *Stats
}

type Option func(c *Collector)

func Dir(dir string) Option              { return func(c *Collector) { c.dir = dir } }
func Location(loc *time.Location) Option { return func(c *Collector) { c.location = loc } }
func Salt(salt string) Option            { return func(c *Collector) { c.salt = salt } }

func New(options ...Option) *Collector {
	c := &Collector{salt: RandomString(32), location: time.Local}
	for _, opt := range options {
		opt(c)
	}
	return c
}

func date(t time.Time) time.Time {
	yyyy, mm, dd := t.Date()
	return time.Date(yyyy, mm, dd, 0, 0, 0, 0, t.Location())
}

func (c *Collector) Hit(hit *Hit) error {
	// TODO: add blacklisting logic
	if filepath.Ext(hit.URI) != "" || strings.Contains(hit.URI, "/_") {
		return nil
	}
	c.Lock()
	defer c.Unlock()
	if err := c.checkAppender(false); err != nil {
		return err
	}
	if startTime := c.appender.StartTime(); date(hit.Timestamp) != date(startTime) && !startTime.IsZero() {
		if err := c.closeAppender(); err != nil {
			return err
		} else if err := c.checkHistoricalStats(); err != nil {
			return err
		} else if stats, err := c.readDailyStats(); err != nil {
			return err
		} else {
			c.mergeAppender(stats)
			if err := c.saveHistoricalStats(); err != nil {
				return err
			} else if err := c.checkAppender(true); err != nil {
				return err
			}
		}
	}
	return c.appender.Append(hit)
}

func (c *Collector) checkHistoricalStats() error {
	if c.history != nil {
		return nil
	}
	b, err := ioutil.ReadFile(filepath.Join(c.dir, historyLog))
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	stats, err := ParseStatsCSV(string(b))
	if err != nil {
		return err
	}
	c.history = stats
	return nil
}

func (c *Collector) saveHistoricalStats() error {
	return ioutil.WriteFile(filepath.Join(c.dir, historyLog), []byte(c.history.CSV()), 0666)
}

func (c *Collector) mergeAppender(daily *Stats) {
	if c.history.Start.IsZero() {
		c.history.Start = date(daily.Start)
	}
	n := int(date(daily.Start).Sub(c.history.Start)/(time.Hour*24)) + 1
	for i, frame := range c.history.frames() {
		frame.Grow(n)
		for _, row := range daily.frames()[i].Rows {
			total := 0
			for _, v := range row.Values {
				total = total + v
			}
			u := frame.Row(row.Name).Values
			u[len(u)-1] = total
		}
	}
}

func (c *Collector) checkAppender(truncate bool) error {
	if c.appender == nil {
		ap, err := NewAppender(filepath.Join(c.dir, dailyLog), truncate)
		if err != nil {
			return err
		}
		c.appender = ap
	}
	return nil
}

func (c *Collector) readDailyStats() (*Stats, error) {
	return ParseAppendLog(filepath.Join(c.dir, dailyLog), c.location)
}

func (c *Collector) Stats() (*Stats, *Stats, error) {
	c.Lock()
	defer c.Unlock()
	if err := c.checkHistoricalStats(); err != nil {
		return nil, nil, err
	} else if daily, err := c.readDailyStats(); err != nil {
		return nil, nil, err
	} else {
		c.mergeAppender(daily)
		// TODO: "Daily"  may actually be old, return empty stats if os
		return daily, c.history, nil
	}
}

func (c *Collector) closeAppender() error {
	if c.appender != nil {
		if err := c.appender.Close(); err != nil {
			return err
		}
		c.appender = nil
	}
	return nil
}

func (c *Collector) Close() error {
	c.Lock()
	defer c.Unlock()
	return c.closeAppender()
}

var gif = []byte{
	0x47, 0x49, 0x46, 0x38, 0x39, 0x61, 0x01, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x21, 0xF9, 0x04,
	0x01, 0x00, 0x00, 0x00, 0x00, 0x2C, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00, 0x02,
}

// ServeHTTP makes Collector implement a http.Handler interface. This handler
// acts as an API and allows to collect stats via a tracking pixel or POST API.
func (c *Collector) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		w.Header().Add("Content-Type", "image/gif")
		w.Header().Set("Tk", "N")
		w.Header().Set("Expires", "Mon, 01 Jan 1990 00:00:00 GMT")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Pragma", "no-cache")
		_, _ = w.Write(gif)
	} else if r.Method == "POST" || r.Method == "PUT" {
		if r.Header.Get("Content-Type") == "application/json" {
			// TODO: parse JSON and put values into query
		}
		w.WriteHeader(http.StatusNoContent)
	}
	_ = c.Hit(hit(r, c.salt, true))
}

// Add allows to collect a hit caused by the given request.
func (c *Collector) Add(r *http.Request) error {
	return c.Hit(hit(r, c.salt, false))
}

// Collect is a middleware that wraps an existing handler and collects every hit/request.
func (c *Collector) Collect(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = c.Add(r)
		h.ServeHTTP(w, r)
	})
}

// Report returns a handler that renders the dashboard report for the collected stats
func (c *Collector) Report() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "text/html")
		html, err := c.ReportHTML()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		if _, err := io.WriteString(w, html); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
}

func (c *Collector) ReportHTML() (string, error) {
	b := &bytes.Buffer{}
	daily, history, err := c.Stats()
	if err == nil {
		err = ReportTemplate.Execute(b, struct {
			Daily   *Stats
			History *Stats
			Map     string
		}{daily, history, mapSVG})
	}
	return b.String(), nil
}

func RandomString(n int) string {
	letters := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

var ReportTemplate = template.Must(template.New("").Parse(reportHTML))

//go:embed "report.html"
var reportHTML string

//go:embed "worldmap.svg"
var mapSVG string
