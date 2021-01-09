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

// String returns a CSV-formatted stats representation.
func (stats *Stats) String() string {
	b := &strings.Builder{}
	// Header: timestamp and interval as a comment
	b.WriteByte('#')
	b.WriteString(stats.Start.Format(time.RFC3339))
	b.WriteByte(',')
	b.WriteString(stats.Interval.String())
	b.WriteByte('\n')
	// Frames
	for _, frame := range stats.frames() {
		for _, row := range frame.rows {
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

// ParseStats returns a Stats instance from the given CSV text string.
func ParseStats(s string) (*Stats, error) {
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
	rows []Row
}

// Len returns the current length of the row time series (i.e. row width)
// within the frame.
func (f *Frame) Len() int {
	if f.len == 0 && len(f.rows) > 0 {
		f.len = len(f.rows[0].Values)
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
	for i := range f.rows {
		row := &f.rows[i]
		if m := size - len(row.Values); m > 0 {
			row.Values = append(row.Values, make([]int, m)...)
		} else {
			row.Values = row.Values[:size]
		}
	}
}

// Rows returns a list of all rows within a frame. The resulting slice should
// be treated as read-only and should never be mutated.
func (f *Frame) Rows() []Row { return f.rows }

// Row returns an existing row by its name or inserts one. It ensures that rows
// are sorted alphabetically withing the data frame.
func (f *Frame) Row(name string) *Row {
	i, found := f.find(name)
	if !found {
		f.rows = append(f.rows, Row{})
		copy(f.rows[i+1:], f.rows[i:])
		f.rows[i].Name = name
		f.rows[i].Values = make([]int, f.Len())
	}
	return &f.rows[i]
}

// Delete removes a row from the frame
func (f *Frame) Delete(name string) bool {
	i, found := f.find(name)
	if found {
		copy(f.rows[i:], f.rows[i+1:])
		f.rows = f.rows[:len(f.rows)-1]
	}
	return found
}

// Find looks up for a row in the frame. It returns an index of the row and a
// flag if such row exists. If the flag is false - the row should be inserted
// at the provided index.
func (f *Frame) find(name string) (int, bool) {
	i, j := 0, len(f.rows)
	for i < j {
		h := int(uint(i+j) >> 1)
		if f.rows[h].Name < name {
			i = h + 1
		} else {
			j = h
		}
	}
	found := i < len(f.rows) && f.rows[i].Name == name
	return i, found
}

// Row is a single record for names time series in the stats data frame. Row
// has a unique name and a series of integer values, one for each time slot of
// the data frame.
type Row struct {
	Name   string
	Values []int
}
