package nullitics

import (
	"errors"
	"strconv"
	"strings"
	"time"
)

// Stats is an aggregated data from the various site-related statistics over a
// given time period.
type Stats struct {
	Start     time.Time
	Interval  time.Duration
	URIs      Frame
	Sessions  Frame
	Refs      Frame
	Countries Frame
	Devices   Frame
}

func (stats *Stats) frames() []*Frame {
	return []*Frame{&stats.URIs, &stats.Sessions, &stats.Refs, &stats.Countries, &stats.Devices}
}

// CSV returns a CSV-formatted text stats representation.
func (stats *Stats) CSV() string {
	b := &strings.Builder{}
	// Header: timestamp and interval as a comment
	b.WriteByte('#')
	b.WriteString(stats.Start.Format(time.RFC3339))
	b.WriteByte(',')
	b.WriteString(stats.Interval.String())
	b.WriteByte('\n')
	// Frames
	for _, frame := range stats.frames() {
		for _, row := range frame.Rows {
			b.WriteString(row.Name)
			for _, v := range row.Values {
				b.WriteByte(',')
				b.WriteString(strconv.Itoa(v))
			}
			b.WriteByte('\n')
		}
		b.WriteByte('\n')
	}
	return b.String()
}

// ParseStatsCSV returns a Stats instance from the given CSV text string.
func ParseStatsCSV(s string) (*Stats, error) {
	lines := strings.Split(s, "\n")
	if lines[0] == "" {
		return &Stats{Interval: time.Hour * 24}, nil
	}
	if lines[0][0] != '#' {
		return nil, errors.New("first line must be a comment")
	}
	parts := strings.Split(lines[0][1:], ",")
	if len(parts) != 2 {
		return nil, errors.New("first line must contain a timestamp and interval")
	}
	t, err := time.Parse(time.RFC3339, parts[0])
	if err != nil {
		return nil, err
	}
	d, err := time.ParseDuration(parts[1])
	if err != nil {
		return nil, err
	}
	stats := &Stats{Start: t, Interval: d}
	lines = lines[1:]
	n := 0
	for _, frame := range stats.frames() {
		frame.Grow(n)
		for len(lines) > 0 {
			line := lines[0]
			lines = lines[1:]
			if line == "" {
				break
			}
			parts := strings.Split(line, ",")
			if len(parts) < 2 {
				return nil, errors.New("expected at least two fields per row")
			}
			n = len(parts) - 1
			frame.Grow(n)
			row := frame.Row(parts[0])
			for i, v := range parts[1:] {
				n, err := strconv.ParseInt(v, 10, 32)
				if err != nil {
					return nil, err
				}
				row.Values[i] = int(n)
			}
		}
	}
	return stats, nil
}

// Frame is a matrix-shaped data frame, used to store time series of named
// integer values.
type Frame struct {
	len  int
	Rows []Row
}

// Len returns the current length of the row time series (i.e. row width)
// within the frame.
func (f *Frame) Len() int {
	if f.len == 0 && len(f.Rows) > 0 {
		f.len = len(f.Rows[0].Values)
	}
	return f.len
}

// Grow resizes all the rows of the frame to the given width. Same width will
// be also applied to the new rows that will be created in this frame.
func (f *Frame) Grow(size int) {
	if size == f.Len() {
		return
	}
	f.len = size
	for i := range f.Rows {
		row := &f.Rows[i]
		if m := size - len(row.Values); m > 0 {
			row.Values = append(row.Values, make([]int, m)...)
		} else {
			row.Values = row.Values[:size]
		}
	}
}

// Row returns an existing row by its name or inserts one. It ensures that rows
// are sorted alphabetically withing the data frame.
func (f *Frame) Row(name string) *Row {
	i, found := f.find(name)
	if !found {
		f.Rows = append(f.Rows, Row{})
		copy(f.Rows[i+1:], f.Rows[i:])
		f.Rows[i].Name = name
		f.Rows[i].Values = make([]int, f.Len())
	}
	return &f.Rows[i]
}

// Delete removes a row from the frame
func (f *Frame) Delete(name string) bool {
	i, found := f.find(name)
	if found {
		copy(f.Rows[i:], f.Rows[i+1:])
		f.Rows = f.Rows[:len(f.Rows)-1]
	}
	return found
}

// Find looks up for a row in the frame. It returns an index of the row and a
// flag if such row exists. If the flag is false - the row should be inserted
// at the provided index.
func (f *Frame) find(name string) (int, bool) {
	i, j := 0, len(f.Rows)
	for i < j {
		h := int(uint(i+j) >> 1)
		if f.Rows[h].Name < name {
			i = h + 1
		} else {
			j = h
		}
	}
	found := i < len(f.Rows) && f.Rows[i].Name == name
	return i, found
}

// Row is a single record for names time series in the stats data frame. Row
// has a unique name and a series of integer values, one for each time slot of
// the data frame.
type Row struct {
	Name   string
	Values []int
}

// Get is a helper method to safely read Row values without taking care about row bounds.
func (r *Row) Get(i int) int {
	if i < 0 || i >= len(r.Values) {
		return 0
	}
	return r.Values[i]
}

func (r *Row) Last(n int) (sum int) {
	for i := 0; i < n; i++ {
		sum = sum + r.Get(len(r.Values)-1-i)
	}
	return sum
}
