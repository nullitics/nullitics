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
	Mobile  = "mobile"
	Desktop = "desktop"
)

var (
	// Request headers, containing the real user IP address
	ipHeaders = []string{"X-Real-IP", "X-Forwarded-For"}
	// User-Agent substrings, typical only for mobile devices
	mobileUAs = []string{"iPhone", "iPad", "Android"}
)

type Hit struct {
	Timestamp time.Time
	URI       string
	Session   string
	Ref       string
	Country   string
	Device    string
}

// Device returns the device type from the request.
// It checks for the "d" query parameter, which would be display width in pixels.
// Devices larger than 992px are considered desktops (break point taken from Bootstrap).
// If "d" parameter is not present - it checks for the User-Agent for certain mobile keywords.
func device(r *http.Request) string {
	if d, _ := strconv.Atoi(r.FormValue("d")); d > 0 {
		if d < 992 {
			return Mobile
		}
		return Desktop
	}
	ua := r.UserAgent()
	for _, mobile := range mobileUAs {
		if strings.Contains(ua, mobile) {
			return Mobile
		}
	}
	return Desktop
}

// IP returns the (most likely) real user IP address. Nullitics does not store
// any of the IP addresses, however they may be used to detect user location
// and identify sessions.
func ip(r *http.Request) string {
	for _, hdr := range ipHeaders {
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
func session(ip string, salt string) string {
	s := ip + date(Now()).Format("20060102") + salt
	hash := md5.Sum([]byte(s))
	return hex.EncodeToString(hash[:4])
}

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

func API(r *http.Request, salt string) *Hit {
	u, err := url.Parse(r.Referer())
	if err != nil {
		return &Hit{}
	}

	ipaddr := ip(r)
	hit := &Hit{Timestamp: Now(), URI: u.Path, Session: session(ipaddr, salt)}

	// Referrer can be passed via "r" of the referrer, or "utm_source" of the actual page
	if ref := r.FormValue("r"); ref != "" {
		hit.Ref = ref
	} else if ref := u.Query().Get("utm_source"); ref != "" {
		hit.Ref = ref
	}
	hit.Country = u.Query().Get("c")
	if hit.Country == "" {
		hit.Country = GeoDB.Find(ipaddr)
		if hit.Country == "" {
			hit.Country = lang(r)
		}
	}
	hit.Device = device(r)
	return hit
}

func Page(r *http.Request, salt string) *Hit {
	path := r.URL.Path
	ref := r.FormValue("utm_source")
	if ref == "" {
		ref = r.Referer()
	}
	ipaddr := ip(r)
	cn := lang(r)
	return &Hit{
		Timestamp: time.Now(),
		URI:       path,
		Session:   session(ipaddr, salt),
		Ref:       ref,
		Country:   cn,
		Device:    device(r),
	}
}
