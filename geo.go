package nullitics

import (
	"archive/zip"
	"bytes"
	"encoding/csv"
	"errors"
	"io"
	"net"
	"os"
	"sort"
	"strings"
)

var GeoDB, _ = NewGeoDB(os.Getenv("GEODB"))

type geodb []ipRange

type ipRange struct {
	Net     *net.IPNet
	Country string
}

func NewGeoDB(zipfile string) (geodb, error) {
	f, err := os.Open(zipfile)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}
	zf, err := zip.NewReader(f, fi.Size())
	if err != nil {
		return nil, err
	}

	var blocks, countries *zip.File
	for _, f := range zf.File {
		if strings.HasSuffix(f.Name, "-Blocks-IPv4.csv") {
			blocks = f
		} else if strings.HasSuffix(f.Name, "-Country-Locations-en.csv") {
			countries = f
		}
	}
	if blocks == nil || countries == nil {
		return nil, errors.New("ZIP does not contains blocks or countries")
	}

	db := geodb{}
	cn := map[string]string{}

	if err := readCSV(countries, []string{"geoname_id", "country_iso_code"}, func(row []string) error {
		cn[row[0]] = row[1]
		return nil
	}); err != nil {
		return nil, err
	}

	if err := readCSV(blocks, []string{"network", "geoname_id"}, func(row []string) error {
		network, id := row[0], row[1]
		country, ok := cn[id]
		if !ok {
			// Some ranges may not contain country data
			return nil
		}
		_, ipnet, err := net.ParseCIDR(network)
		if err != nil {
			return err
		}
		db = append(db, ipRange{Net: ipnet, Country: country})
		return nil
	}); err != nil {
		return nil, err
	}
	return db, nil
}

func readCSV(file *zip.File, fields []string, f func([]string) error) error {
	r, err := file.Open()
	if err != nil {
		return err
	}
	defer r.Close()
	rows := csv.NewReader(r)
	header, err := rows.Read()
	if err != nil {
		return err
	}
	indices := make([]int, len(fields))
	for i, f := range fields {
		found := false
		for j, h := range header {
			if h == f {
				indices[i] = j
				found = true
			}
		}
		if !found {
			return errors.New("missing field: " + f)
		}
	}
	cols := make([]string, len(fields))
	for {
		row, err := rows.Read()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		for i, j := range indices {
			cols[i] = row[j]
		}
		if err := f(cols); err != nil {
			return err
		}
	}
}

func (db geodb) Find(ipv4 string) string {
	ip := net.ParseIP(ipv4).To4()
	if ip == nil {
		return ""
	}
	i := sort.Search(len(db), func(i int) bool {
		return bytes.Compare(db[i].Net.IP, ip) > 0 || db[i].Net.Contains(ip)
	})
	if i < len(db) && db[i].Net.Contains(ip) {
		return db[i].Country
	}
	return ""
}
