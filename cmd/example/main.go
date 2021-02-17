// This is a very minimal example of using nullitics as a library.
// Try:
//   PORT=8080 go run main.go
// Then open a few pages at http://127.0.0.1:8008/
// Finally, open http://127.0.0.1:8080/_/stats/ and enjoy the statistics report!

package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/nullitics/nullitics"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	c := nullitics.New()

	head := `<head><meta name="viewport" content="width=device-width, initial-scale=1">
	<style>body{margin:auto;max-width:40rem;line-height:1.6;padding:1.6rem;}</style></head>`

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, head+`<h1>A Really Small Collection of Haiku</h1>
		<ul><li><a href="/kerouac">Jack Kerouac</a></li>
		<li><a href="/basho">Matsuo Bashō</a></li></ul>`)
	})
	mux.HandleFunc("/kerouac", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, head+`<h1>Jack Kerouac</h1>
		<p>The bottom of my shoes<br/>are clean<br/>from walking in the rain</p>
		<a href="/">← Back</a>`)
	})
	mux.HandleFunc("/basho", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, head+`<h1>Matsuo Bashō</h1>
		<p>Old pond<br/>frog jumped in<br/>sound of water</p>
		<a href="/">← Back</a>`)
	})

	mux.Handle("/_/stats/", c.Report(nil))

	log.Println("Started on port", port)
	log.Fatal(http.ListenAndServe(":"+port, c.Collect(mux)))
}
