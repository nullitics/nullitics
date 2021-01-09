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

	"github.com/zserge/nullitics"
)

var gif = []byte{
	0x47, 0x49, 0x46, 0x38, 0x39, 0x61, 0x01, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x21, 0xF9, 0x04,
	0x01, 0x00, 0x00, 0x00, 0x00, 0x2C, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00, 0x02,
}

func main() {
	defaultPort := os.Getenv("PORT")
	if defaultPort == "" {
		defaultPort = "8080"
	}
	port := flag.String("port", defaultPort, "Port number")
	addr := flag.String("addr", "http://127.0.0.1:"+defaultPort, "External address of this service")
	dir := flag.String("dir", "", "Directory to store stats")
	loc := flag.String("loc", "Local", "Time zone")
	flag.Parse()

	location, err := time.LoadLocation(*loc)
	if err != nil {
		log.Fatal(err)
	}

	c := nullitics.NewCollector(*dir, location)
	report := c.Report()

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
			w.Header().Add("Content-Type", "image/gif")
			w.Header().Set("Tk", "N")
			w.Header().Set("Expires", "Mon, 01 Jan 1990 00:00:00 GMT")
			w.Header().Set("Cache-Control", "no-store")
			w.Header().Set("Pragma", "no-cache")
			_, _ = w.Write(gif)

			_ = c.Add(nullitics.API(r, nullitics.DefaultSalt))
		}
	})

	log.Println("Started on port " + *port + ", check " + path.Join(*addr+"/stats/"))
	log.Fatal(http.ListenAndServe(":"+*port, nil))
}
