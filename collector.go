package nullitics

import (
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"text/template"
	"time"
)

var dailyLog = "log.csv"
var historyLog = "stats.csv"

type Collector struct {
	sync.Mutex
	dir      string
	location *time.Location
	appender *Appender
	history  *Stats
}

type Option func(c *Collector)

func Dir(dir string) Option              { return func(c *Collector) { c.dir = dir } }
func Location(loc *time.Location) Option { return func(c *Collector) { c.location = loc } }

func New(options ...Option) *Collector {
	c := &Collector{}
	for _, opt := range options {
		opt(c)
	}
	return c
}

func date(t time.Time) time.Time {
	yyyy, mm, dd := t.Date()
	return time.Date(yyyy, mm, dd, 0, 0, 0, 0, t.Location())
}

func (c *Collector) Add(hit *Hit) error {
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
	stats, err := ParseStats(string(b))
	if err != nil {
		return err
	}
	c.history = stats
	return nil
}

func (c *Collector) saveHistoricalStats() error {
	return ioutil.WriteFile(filepath.Join(c.dir, historyLog), []byte(c.history.String()), 0666)
}

func (c *Collector) mergeAppender(daily *Stats) {
	if c.history.Start.IsZero() {
		c.history.Start = date(daily.Start)
	}
	n := int(date(daily.Start).Sub(c.history.Start)/(time.Hour*24)) + 1
	for i, frame := range c.history.frames() {
		frame.Grow(n)
		for _, row := range daily.frames()[i].Rows() {
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

func (c *Collector) Report() http.Handler {
	tmpl := template.Must(template.New("").Parse(html))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		daily, history, err := c.Stats()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err := tmpl.Execute(w, struct{ Daily, History *Stats }{daily, history}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
}

func randomString(n int) string {
	letters := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

var DefaultCollector = New(Dir("nullitics"), Location(time.Local))
var DefaultSalt = randomString(32)

var Now = time.Now

func Collect(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = DefaultCollector.Add(Page(r, DefaultSalt))
		h.ServeHTTP(w, r)
	})
}

func Report() http.Handler { return DefaultCollector.Report() }

var html = `<!doctype html><html>
<head>
</head>
<body>
<h1>Today {{ .Daily.Start.Format "2006-01-02"}}</h1>
<pre>
{{ .Daily }} 
</pre>
<h1>Since {{ .History.Start.Format "2006-01-02"}}</h1>
<pre>
{{ .History }} 
</pre>
</body>
</html>`
