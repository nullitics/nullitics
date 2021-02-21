package nullitics

import (
	"crypto/md5"
	"encoding/hex"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	// Mobile device type
	Mobile = "mobile"
	// Desktop device type
	Desktop = "desktop"
)

var (
	// IPHeaders are request headers, containing the real user IP address
	IPHeaders = []string{"X-Real-IP", "X-Forwarded-For"}
	// MobileUAs are user-agent substrings, typical only for mobile devices
	MobileUAs = []string{"iPhone", "iPad", "Android"}
	// MobileBreakpoint is the maximum screen width for mobile devices, 992px is taken from Bootstrap.
	MobileBreakpoint = 992
	// SkipSubdomains is a list of common subdomains to skip in referrers
	SkipSubdomains = []string{"www.", "www1.", "www2.", "www3.", "www4.", "m.", "l.", "lm.", "i.", "old."}
	// BotAgents is a list of substrings commonly met in bot/crawler User-Agent strings
	BotAgents = []string{"bot", "crawler", "spider", "spyder", "search", "worm", "fetch", "nutch", "http://", "https://"}
)

// Hit is a basic data type describing a single page visit or event.
type Hit struct {
	Timestamp time.Time
	URI       string
	Session   string
	Ref       string
	Country   string
	Device    string
}

func isMobileUserAgent(ua string) bool {
	for _, mobile := range MobileUAs {
		if strings.Contains(ua, mobile) {
			return true
		}
	}
	return false
}

func isMobileScreen(size string) bool {
	d, _ := strconv.Atoi(size)
	return d > 0 && d < MobileBreakpoint
}

// IPAddr returns the (most likely) real user IP address. Nullitics does not store
// any of the IP addresses, however they may be used to detect user location
// and identify sessions.
func ipaddr(r *http.Request) string {
	for _, hdr := range IPHeaders {
		if fields := strings.Fields(r.Header.Get(hdr)); len(fields) > 0 {
			if ip := strings.TrimRight(fields[0], ","); ip != "" {
				return ip
			}
		}
	}
	ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	return ip
}

// Session returns a hash of the user IP address, user agent, current date and
// a salt string. It is unique enough for most typical cases, and does not
// violate user's privacy since no personal data is stored within a session.
func session(ip, ua, salt string) string {
	s := ip + date(Now()).Format("20060102") + ua + salt
	hash := md5.Sum([]byte(s))
	return hex.EncodeToString(hash[:4])
}

// Lang returns country/language code from the request Accept-Language header.
func lang(r *http.Request) string {
	for _, lang := range strings.Split(r.Header.Get("Accept-Language"), ",") {
		if !strings.Contains(lang, ";") {
			if len(lang) == 2 {
				return strings.ToUpper(lang)
			} else if len(lang) == 5 && lang[2] == '-' {
				return strings.ToUpper(lang[3:])
			}
		}
	}
	return ""
}

func validateURI(uri string) string {
	if uri == "" {
		return "/"
	}
	if len(uri) > MaxPathLength {
		return uri[:MaxPathLength-1]
	}
	return uri
}

func isBot(ua string) bool {
	s := strings.ToLower(ua)
	for _, b := range BotAgents {
		if strings.Contains(s, b) {
			return true
		}
	}
	return false
}

func validateRef(ref string) string {
	u, err := url.Parse(ref)
	if err != nil {
		return ""
	}
	host := u.Hostname()
	for _, sub := range SkipSubdomains {
		if strings.HasPrefix(host, sub) {
			host = strings.TrimPrefix(host, sub)
			break
		}
	}
	if strings.HasSuffix(host, ".google.com") ||
		strings.HasPrefix(host, "google.co.") ||
		strings.HasPrefix(host, "google.com.") ||
		(len(host) < 13 && strings.HasPrefix(host, "google.")) ||
		host == "com.google.android.googlequicksearchbox" ||
		host == "com.google.android.gm" {
		host = "google.com"
	} else if strings.HasSuffix(host, ".bing.com") {
		host = "bing.com"
	} else if strings.HasSuffix(host, ".duckduckgo.com") {
		host = "duckduckgo.com"
	} else if strings.HasSuffix(host, ".reddit.com") {
		host = "reddit.com"
	} else if strings.HasSuffix(host, ".wikipedia.org") {
		host = "wikipedia.org"
	}
	if len(host) > MaxRefLength {
		host = host[:MaxRefLength-1]
	}
	return host
}

func hit(r *http.Request, salt string, api bool) *Hit {
	// TODO: handle DNT
	// Create a hit object with current timestamp
	hit := &Hit{Timestamp: Now()}
	// Skip bots
	if isBot(r.UserAgent()) {
		return hit
	}
	// If collector is used as a middleware - use request Path, otherwise use r.Referer path
	if api {
		u, err := url.Parse(r.URL.Query().Get("u"))
		if u == nil || u.String() == "" || err != nil {
			u, err = url.Parse(r.Referer())
		}
		if err != nil {
			return hit
		}
		hit.URI = u.Path
		hit.Ref = u.Query().Get("utm_source")
	} else {
		hit.URI = r.URL.Path
		hit.Ref = r.URL.Query().Get("utm_source")
	}
	// Validate URI
	hit.URI = validateURI(hit.URI)
	// Create Session hash
	ip := ipaddr(r)
	hit.Session = session(ip, r.UserAgent(), salt)
	// Fill referrer and validate its value
	if api && hit.Ref == "" {
		hit.Ref = r.FormValue("r")
	}
	hit.Ref = validateRef(hit.Ref)
	// Get device type via API parameters or via user agent
	if (api && isMobileScreen(r.FormValue("d"))) || isMobileUserAgent(r.UserAgent()) {
		hit.Device = Mobile
	} else {
		hit.Device = Desktop
	}
	// Get ISO country code from IP address (if possible), or from Accept-Language header
	if cn := r.FormValue("c"); api && cn != "" {
		hit.Country = cn
	} else if cn := GeoDB.Find(ip); cn != "" {
		hit.Country = cn
	} else {
		hit.Country = lang(r)
	}
	return hit
}
