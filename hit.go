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

type Hit struct {
	Timestamp time.Time
	URI       string
	Session   string
	Ref       string
	Country   string
	Device    string
}

func device(r *http.Request) string {
	if d, _ := strconv.Atoi(r.URL.Query().Get("d")); d > 0 {
		if d < 992 {
			return "mobile"
		}
		return "desktop"
	}
	ua := r.UserAgent()
	if strings.Contains(ua, "iPhone") || strings.Contains(ua, "iPad") || strings.Contains(ua, "Android") {
		return "mobile"
	}
	return "desktop"
}

func ip(r *http.Request) string {
	for _, hdr := range []string{"X-Real-IP", "X-Forwarded-For", "CF-Connecting-IP"} {
		if fields := strings.Fields(r.Header.Get(hdr)); len(fields) > 0 {
			if ip := strings.TrimRight(fields[0], ","); ip != "" {
				return ip
			}
		}
	}
	ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	return ip
}

func session(r *http.Request, salt string) string {
	s := ip(r) + date(Now()).Format("20060102") + salt
	hash := md5.Sum([]byte(s))
	return hex.EncodeToString(hash[:4])
}

func API(r *http.Request, salt string) *Hit {
	u, err := url.Parse(r.Referer())
	if err != nil {
		return &Hit{}
	}

	hit := &Hit{Timestamp: Now(), URI: u.Path, Session: session(r, salt)}

	// Referrer can be passed via "r" of the referrer, or "utm_source" of the actual page
	if ref := r.URL.Query().Get("r"); ref != "" {
		hit.Ref = ref
	} else if ref := u.Query().Get("utm_source"); ref != "" {
		hit.Ref = ref
	}
	hit.Country = u.Query().Get("c")
	hit.Device = device(r)
	return hit
}

func Page(r *http.Request, salt string) *Hit {
	path := r.URL.Path
	ref := r.URL.Query().Get("utm_source")
	if ref == "" {
		ref = r.Referer()
	}
	cn := "" // TODO: use from IP
	return &Hit{
		Timestamp: time.Now(),
		URI:       path,
		Session:   session(r, salt),
		Ref:       ref,
		Country:   cn,
		Device:    device(r),
	}
}
