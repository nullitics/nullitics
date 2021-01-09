package nullitics

import (
	"math/rand"
	"reflect"
	"testing"
	"time"
)

// Ensure that search within a sorted frame works as expected
func TestFrameFind(t *testing.T) {
	f := Frame{rows: []Row{{Name: "a1"}, {Name: "a4"}, {Name: "a7"}}}
	for _, test := range []struct {
		name  string
		index int
		found bool
	}{
		{"a1", 0, true},
		{"a2", 1, false},
		{"a3", 1, false},
		{"a4", 1, true},
		{"a5", 2, false},
		{"a6", 2, false},
		{"a7", 2, true},
		{"a8", 3, false},
	} {
		i, ok := f.find(test.name)
		if i != test.index || ok != test.found {
			t.Error(test, i, ok)
		}
	}
}

// Ensure that frame insertions and removals are working as expected
func TestFrameInsertDelete(t *testing.T) {
	f := Frame{}
	f.Row("foo")
	f.Row("bar")
	f.Row("baz")
	rows := f.Rows()
	if len(rows) != 3 || rows[0].Name != "bar" || rows[1].Name != "baz" || rows[2].Name != "foo" {
		t.Error(rows)
	}
	f.Row("qux")
	if ok := f.Delete("baz"); !ok {
		t.Error()
	}
	if ok := f.Delete("invalid_row"); ok {
		t.Error()
	}
	rows = f.Rows()
	if len(rows) != 3 || rows[0].Name != "bar" || rows[1].Name != "foo" || rows[2].Name != "qux" {
		t.Error(rows)
	}
}

// Ensure that frame grows and shrinks and keeps its values
func TestFrameGrow(t *testing.T) {
	f := Frame{}
	if n := f.Len(); n != 0 {
		t.Error(n)
	}
	f.Grow(5)
	if n := f.Len(); n != 5 {
		t.Error(n)
	}
	f.Row("bar")
	f.Row("qux")
	row := f.Row("foo")
	if f.Len() != 5 || len(row.Values) != 5 {
		t.Error(row.Values)
	}
	copy(row.Values, []int{1, 2, 3, 4, 5})
	f.Grow(7)
	row = f.Row("foo")
	if len(row.Values) != 7 || row.Values[4] != 5 || row.Values[5] != 0 || row.Values[6] != 0 {
		t.Error(row.Values)
	}
	f.Grow(3)
	row = f.Row("foo")
	if len(row.Values) != 3 || row.Values[0] != 1 || row.Values[1] != 2 || row.Values[2] != 3 {
		t.Error(row.Values)
	}
	if n := f.Len(); n != 3 {
		t.Error(n)
	}
}

func TestStats(t *testing.T) {
	stats := &Stats{Start: Now().Round(time.Second), Interval: 3 * time.Hour}
	for _, f := range stats.frames() {
		f.Grow(30)
		for i := 0; i < rand.Intn(4); i++ {
			row := f.Row(randomString(8))
			for i := range row.Values {
				row.Values[i] = rand.Intn(100)
			}
		}
	}
	s := stats.String()
	parsed, err := ParseStats(s)
	if err != nil {
		t.Error(err)
	}
	if stats.Start.Equal(parsed.Start) == false || stats.Interval != parsed.Interval {
		t.Error(stats.Start, parsed.Start, stats.Interval, parsed.Interval)
	}
	for i, f := range stats.frames() {
		if reflect.DeepEqual(f, parsed.frames()[i]) == false {
			t.Error(f, parsed.frames()[i])
		}
	}
}
