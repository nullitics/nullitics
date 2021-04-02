package nullitics

import (
	_ "embed" // embed package must be imported for embedded FS to work
	"fmt"
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

// Now is a timestamp generator. It is made public so that the caller could
// mock it or replace with their own implementation.
var Now = time.Now

var dailyLog = "log.csv"
var historyLog = "stats.csv"

var (
	// MaxPathLength is the longest possible URI or event name length.
	MaxPathLength = 200
	// MaxRefLength is the longest possible referrer length. Typically, domain
	// names are no longer than 63 bytes.
	MaxRefLength = 64
	// MaxCountryLength is the longest possible country code. Nullitics uses ISO
	// codes, so 2 bytes should be enough.
	MaxCountryLength = 2
)

// Collector is an abstracton that records Hits and provides collected Stats.
type Collector struct {
	sync.Mutex
	dir      string
	location *time.Location
	salt     string
	appender *Appender
	history  *Stats
}

// Option is a function option data type for Collector.
type Option func(c *Collector)

// Dir sets the collector working directory.
func Dir(dir string) Option { return func(c *Collector) { c.dir = dir } }

// Location set the collector time zone.
func Location(loc *time.Location) Option { return func(c *Collector) { c.location = loc } }

// Salt initializes the collector salt for hashes. By default the salt is a random string.
func Salt(salt string) Option { return func(c *Collector) { c.salt = salt } }

// New creates a collector instance with the given options.
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

// Hit records a single hit data.
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
	startTime := c.appender.StartTime().In(c.location)
	if date(hit.Timestamp.In(c.location)) != date(startTime) && !startTime.IsZero() {
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

// Stats returns the daily and overall statistic for the given collector. Daily
// stats have hourly precision, total stats have daily precision.
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

// Close shuts down the collector.
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

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(status int) { rw.status = status }
func (rw *responseWriter) Write(b []byte) (int, error) {
	if rw.status == 0 {
		rw.WriteHeader(http.StatusOK)
	}
	return rw.ResponseWriter.Write(b)
}

// Collect is a middleware that wraps an existing handler and collects every hit/request.
func (c *Collector) Collect(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rw := &responseWriter{w, 0}
		h.ServeHTTP(rw, r)
		if rw.status >= 200 && rw.status < 300 {
			_ = c.Add(r)
		}
	})
}

// Report returns a handler that renders the dashboard report for the collected stats
func (c *Collector) Report(extra interface{}) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "text/html")
		if err := c.report(w, extra); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
}

func (c *Collector) report(w io.Writer, extra interface{}) error {
	daily, history, err := c.Stats()
	if err != nil {
		return err
	}
	return ReportTemplate.Execute(w, struct {
		Daily   *Stats
		History *Stats
		Extra   interface{}
	}{daily, history, extra})
}

// RandomString is a helper utility to generate random string IDs, salts etc.
func RandomString(n int) string {
	letters := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

var (
	//go:embed report/report.html
	reportHTML string
	//go:embed report/report.js
	reportJS string
	//go:embed report/worldmap.svg
	worldMapSVG string

	fullTemplate = fmt.Sprintf(`
		{{define "script"}}
		<script type="text/javascript">
		const worldMapSVG = %q;
		const fullData = {{ .History }};
		const dailyData = {{ .Daily }};
		%s
		</script>
		{{end}}`, worldMapSVG, reportJS) + reportHTML

	// ReportTemplate is a http/template.Template for the default dashboard UI.
	// Feel free to customize it to your own needs by providing the {{head}},
	// {{extra_head}}, {{header}} or {{footer}} sections.
	ReportTemplate = template.Must(template.New("").Parse(fullTemplate))
)
