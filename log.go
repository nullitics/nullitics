package nullitics

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Appender is an append-only log writer.
type Appender struct {
	f     *os.File
	start time.Time
	sb    strings.Builder
}

// NewAppender creates an Appender for the given log filename. It may
// optionally truncate the log file before using it.
func NewAppender(filename string, truncate bool) (*Appender, error) {
	flags := os.O_RDWR | os.O_CREATE
	if truncate {
		flags = flags | os.O_TRUNC
	}
	_ = os.MkdirAll(filepath.Dir(filename), 0777)
	f, err := os.OpenFile(filename, flags, 0666)
	if err != nil {
		return nil, err
	}

	// Read timestamp, if any. Start time is zero if the log file is empty
	buf := make([]byte, 64)
	n, err := f.Read(buf)
	if n == 0 && err == io.EOF {
		return &Appender{f: f}, nil
	} else if err != nil {
		return nil, err
	}
	unix := int64(0)
	for i := 0; i < n; i++ {
		if buf[i] >= '0' && buf[i] <= '9' {
			unix = unix*10 + int64(buf[i]-'0')
		} else {
			break
		}
	}

	// Jump to the end of the file for appending
	if _, err := f.Seek(0, io.SeekEnd); err != nil {
		return nil, err
	}

	return &Appender{f: f, start: time.Unix(unix, 0)}, nil
}

// StartTime returns the timestamp of the first hit in the log.
func (ap *Appender) StartTime() time.Time { return ap.start }

// Close shuts down the appender.
func (ap *Appender) Close() error { return ap.f.Close() }

// Append write hit data to the end of the log file.
func (ap *Appender) Append(hit *Hit) error {
	ap.sb.Reset()
	ap.sb.WriteString(strconv.FormatInt(hit.Timestamp.Unix(), 10))
	ap.sb.WriteByte(',')
	ap.sb.WriteString(hit.URI)
	ap.sb.WriteByte(',')
	ap.sb.WriteString(hit.Session)
	ap.sb.WriteByte(',')
	ap.sb.WriteString(hit.Ref)
	ap.sb.WriteByte(',')
	ap.sb.WriteString(hit.Country)
	ap.sb.WriteByte(',')
	ap.sb.WriteString(hit.Device)
	ap.sb.WriteByte('\n')
	_, err := ap.f.Write([]byte(ap.sb.String()))
	if err == nil && ap.start.IsZero() {
		ap.start = hit.Timestamp
	}
	return err
}

// ParseAppendLog read the log file, assuming the timestamps are in the given
// time zone, and returns a Stats object with hourly precision.
func ParseAppendLog(filename string, location *time.Location) (*Stats, error) {
	stats := &Stats{
		Interval:  time.Hour,
		URIs:      Frame{len: 24},
		Sessions:  Frame{len: 24},
		Refs:      Frame{len: 24},
		Countries: Frame{len: 24},
		Devices:   Frame{len: 24},
	}
	f, err := os.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return stats, nil
		}
		return nil, err
	}
	defer f.Close()
	r := bufio.NewReader(f)
	sessions := map[string]bool{}
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				return nil, err
			}
			break
		}
		if len(line) > 0 && line[len(line)-1] == '\n' {
			line = line[:len(line)-1]
		}
		parts := strings.Split(line, ",")
		if len(parts) != 6 {
			continue
		}
		unix, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			continue
		}
		timestamp := time.Unix(unix, 0).In(location)
		if stats.Start.IsZero() {
			stats.Start = date(timestamp)
		}
		hour := timestamp.Hour()
		if uri := parts[1]; uri != "" {
			stats.URIs.Row(uri).Values[hour]++
		}
		if sess := parts[2]; sess == "" || !sessions[sess] {
			sessions[sess] = true
			stats.Sessions.Row("sessions").Values[hour]++
			if ref := parts[3]; ref != "" {
				stats.Refs.Row(ref).Values[hour]++
			}
			if cn := parts[4]; cn != "" {
				stats.Countries.Row(cn).Values[hour]++
			}
			if dev := parts[5]; dev != "" {
				stats.Devices.Row(dev).Values[hour]++
			}
		}
	}
	return stats, nil
}
