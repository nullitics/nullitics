# NULLITICS

[![Build Status](https://img.shields.io/github/workflow/status/nullitics/nullitics/CI%20Pipeline)](https://github.com/nullitics/nullitics)
[![GoDoc](https://godoc.org/github.com/nullitics/nullitics?status.svg)](https://godoc.org/github.com/nullitics/nullitics)
[![GoReportCard example](https://goreportcard.com/badge/github.com/nullitics/nullitics)](https://goreportcard.com/report/github.com/nullitics/nullitics)
[![Docker Image Size (latest by date)](https://img.shields.io/docker/image-size/zserge/nullitics)](https://hub.docker.com/repository/docker/zserge/nullitics/)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)
[![Buy me a coffee](https://img.shields.io/badge/buymeacoffee-donate-yellow.svg)](https://buymeacoffee.com/zserge)

Zero-effort web analytics. This is a self-hosted open-source version of Nullitics. Check out https://nullitics.com for the world cheapest web analytics in the cloud.

## Features

* Privacy-focused (no cookies, fully anonymized).
* Easy to set up (no databases, no external dependencies).
* Meaningful, stylish [dashboard](https://nullitics.com/dashboard/zserge.com).
* Easy to understand metrics (unique visitors, page views, referrers, countries, device types).
* Very lightweight (Docker image is under 10MB, same is the size of the executable).
* Compliant with GDPR, ePrivacy, PECR, CCPA, and COPPA.
* Fast (can handle 35K req/sec on my humble personal server).
* By-pass ad-blockers.
* Can be used as a standalone service or as a Go library.
* You own your data.

## How to use it?

Nullitics comes in several flavours:

* Embedded (a Go library to use in your Go projects).
* Self-hosted (a static binary or a Docker container to run on your server).
* Cloud (the world cheapest web analytics, that also respect your users' privacy).

### Library API

The package API is designed to be as simple as possible:

```go
mux := http.NewServeMux()
...
// Create a new Nullitics collector to collect analytics
c := nullitics.New()
// Register a report endpoint to see the dashboard
mux.Handle("/_/stats/", c.Report(nil))
// Wrap mux so that every request would be recorded
http.ListenAndServe(":"+port, c.Collect(mux))
```

Of course, there's plenty of room for customization, see [GoDoc](https://godoc.org/github.com/nullitics/nullitics) for further details.

Also you may try out the `./cmd/example` to see how Nullitics work as library.

### Self-hosted container

Running container on a personal server is even simpler:

```
docker run -p 8080:8080 zserge/nullitics \
	-url mydomain.com \
	-dir nullitics-data \
	-loc Europe/Berlin \
```

You may check `./cmd/pixel` to see how the standalone version works.

Of course, you can still build it yourself and run as a Linux service instead of a Docker container, if you like.

## License

Code is distributed under MIT license, feel free to use it in your proprietary projects as well.
