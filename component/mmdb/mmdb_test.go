package mmdb

import (
	"fmt"
	"net/netip"
	"testing"
)

func TestLoadMMDB(t *testing.T) {
	path := "/Users/jimyag/src/work/github/mirror-proxy/GeoLite2-Country.mmdb"
	mmdb, err := LoadMMDB(path)
	if err != nil {
		t.Fatal(err)
	}
	var record struct {
		Country struct {
			ISOCode string `maxminddb:"iso_code"`
		} `maxminddb:"country"`
	}
	ip := netip.MustParseAddr("139.226.59.153")
	result := mmdb.Lookup(ip)
	err = result.Decode(&record)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(record.Country.ISOCode)
}
