package nullitics

import (
	"io/ioutil"
	"os"
	"testing"
	"time"
)

func TestAppender(t *testing.T) {
	testFile := "_test_append.log"
	timestamp := time.Unix(123456789, 0)
	defer os.Remove(testFile)

	// Ensure file does not exist before the tests
	os.Remove(testFile)

	t.Run("New", func(t *testing.T) {
		ap, err := NewAppender(testFile, false)
		if err != nil {
			t.Fatal(err)
		}
		// Time for empty log files should be zero
		if st := ap.StartTime(); !st.IsZero() {
			t.Error(st)
		}
		// File should be created and have zero size
		if fi, err := os.Stat(testFile); err != nil || fi.Size() != 0 {
			t.Error(fi, err)
		}
		// Write one record
		if err := ap.Append(&Hit{Timestamp: timestamp, URI: "/foo"}); err != nil {
			t.Error(err)
		}
		if err := ap.Close(); err != nil {
			t.Error(err)
		}
		// File size should be non-zero
		if fi, err := os.Stat(testFile); err != nil || fi.Size() < 15 {
			t.Error(fi, err)
		}
	})

	t.Run("Reopen", func(t *testing.T) {
		ap, err := NewAppender(testFile, false)
		if err != nil {
			t.Fatal(err)
		}
		// Start time should be the timestamp of the first record
		if st := ap.StartTime(); st != timestamp {
			t.Error(st)
		}
		// Write another record
		if err := ap.Append(&Hit{Timestamp: timestamp.Add(time.Second), URI: "/hello"}); err != nil {
			t.Error(err)
		}
		if err := ap.Close(); err != nil {
			t.Error(err)
		}
		b, _ := ioutil.ReadFile(testFile)
		if string(b) != "123456789,/foo,,,,\n123456790,/hello,,,,\n" {
			t.Error(string(b))
		}
	})

	t.Run("Truncate", func(t *testing.T) {
		ap, err := NewAppender(testFile, true)
		if err != nil {
			t.Fatal(err)
		}
		// Start time should be zero again
		if st := ap.StartTime(); !st.IsZero() {
			t.Error(st)
		}
		if err := ap.Close(); err != nil {
			t.Error(err)
		}
		// File size should be non-zero
		if fi, err := os.Stat(testFile); err != nil || fi.Size() != 0 {
			t.Error(fi, err)
		}
	})
}
func TestLogStats(t *testing.T) {
	os.RemoveAll("_testdir")
	c := New(Dir("_testdir"), Location(time.UTC))
	ts := func(s string) time.Time {
		t, _ := time.Parse("2006-01-02 15:04", s)
		return t
	}
	for _, hit := range []*Hit{
		{ts("2021-01-01 10:30"), "/a", "", "", "", ""},
		{ts("2021-01-01 10:42"), "/b", "", "", "", ""},
		{ts("2021-01-01 12:00"), "/c", "", "", "", ""},
		{ts("2021-01-01 18:00"), "/c", "", "", "", ""},
		{ts("2021-01-01 18:01"), "/c", "", "", "", ""},
		{ts("2021-01-01 18:59"), "/d", "", "", "", ""},
		{ts("2021-01-02 00:00"), "/e", "", "", "", ""},
		{ts("2021-01-02 08:00"), "/f", "", "", "", ""},
		{ts("2021-01-04 09:45"), "/g", "", "", "", ""},
		{ts("2021-01-05 07:30"), "/h", "", "", "", ""},
	} {
		if err := c.Add(hit); err != nil {
			t.Error(err)
		}
	}
	d, h, err := c.Stats()
	if err != nil {
		t.Error(err)
	}
	if err := c.Add(&Hit{ts("2021-01-05 00:01"), "/g", "", "", "", ""}); err != nil {
		t.Error(err)
	}
	if err := c.Add(&Hit{ts("2021-01-05 23:59"), "/h", "", "", "", ""}); err != nil {
		t.Error(err)
	}
	d2, h2, err := c.Stats()
	if err != nil {
		t.Error(err)
	}
	// TODO
	//t.Log(d)
	//t.Log(d2)
	//t.Log(h)
	//t.Log(h2)
	_, _, _, _ = d, d2, h, h2
}
