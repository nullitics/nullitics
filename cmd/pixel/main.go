// This is basically "tracking-pixel-as-a-service".
// It serves a blank 1x1px GIF and records how many times it has been called.
// You may use it with a static web site or from other web services.

package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/nullitics/nullitics"
)

func main() {
	defaultPort := os.Getenv("PORT")
	if defaultPort == "" {
		defaultPort = "8080"
	}
	port := flag.String("port", defaultPort, "Port number")
	addr := flag.String("addr", "http://127.0.0.1:"+defaultPort, "External address of this service")
	dir := flag.String("dir", "", "Directory to store stats")
	loc := flag.String("loc", "Local", "Time zone")
	salt := flag.String("salt", nullitics.RandomString(32), "Salt for hashes")
	flag.Parse()

	location, err := time.LoadLocation(*loc)
	if err != nil {
		log.Fatal(err)
	}

	c := nullitics.New(nullitics.Dir(*dir),
		nullitics.Location(location),
		nullitics.Salt(*salt))
	report := c.Report(nil)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Println(r.Method, r.URL.Path, r.UserAgent(), r.Referer())
		switch {
		case strings.HasSuffix(r.URL.Path, "/stats/"):
			// Show statistics report
			report.ServeHTTP(w, r)
		case strings.HasSuffix(r.URL.Path, ".js"):
			// Return a JS snippet
			w.Header().Add("Content-Type", "application/javascript")
			fmt.Fprintf(w, `new Image().src='`+*addr+`/null.gif?r='+encodeURI(document.referrer)+'&d='+screen.width`)
		case strings.HasSuffix(r.URL.Path, ".gif"):
			// Serve a tracking pixel and record a hit
			c.ServeHTTP(w, r)
		}
	})

	log.Println("Started on port " + *port + ", check " + path.Join(*addr+"/stats/"))
	log.Fatal(http.ListenAndServe(":"+*port, nil))
}
