package constant

import (
	"net/netip"
)

type Metadata struct {
	SrcIP    netip.Addr
	Host     string
	Protocol string
}

type MMDBRecord struct {
	Country struct {
		ISOCode string `maxminddb:"iso_code"`
	} `maxminddb:"country"`
}
