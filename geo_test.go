package nullitics

import (
	"os"
	"testing"
)

func TestGeoDB(t *testing.T) {
	geodbFile := os.Getenv("GEODB")
	if geodbFile == "" {
		t.Skip()
	}
	db, err := NewGeoDB(geodbFile)
	if err != nil {
		t.Fatal(err)
	}
	cn := db.Find("8.8.8.8")
	t.Log(cn)
	cn = db.Find("127.0.0.1")
	t.Log(cn)
}
