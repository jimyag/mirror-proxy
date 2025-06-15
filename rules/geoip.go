package rules

import (
	"context"
	"log/slog"
	"net/netip"
	"slices"
	"strings"

	"github.com/oschwald/maxminddb-golang/v2"

	"github.com/jimyag/mirror-proxy/component/mmdb"
	"github.com/jimyag/mirror-proxy/component/resolver"
	"github.com/jimyag/mirror-proxy/config"
	"github.com/jimyag/mirror-proxy/constant"
)

type GeoIPRule struct {
	mmdb    *maxminddb.Reader
	country string
	action  constant.RuleAction
	isSrc   bool
}

var _ Rule = (*GeoIPRule)(nil)

func (g *GeoIPRule) isLan(ip netip.Addr) bool {
	if ip.IsPrivate() ||
		ip.IsUnspecified() ||
		ip.IsLoopback() ||
		ip.IsMulticast() ||
		ip.IsLinkLocalUnicast() {
		return true
	}
	// 只检查 IPv4 地址
	if !ip.Is4() {
		return false
	}
	ipv4 := ip.As4()
	// 检查 100.64.0.0/10 网段 (CGNAT)
	if ipv4[0] == 100 && ipv4[1] >= 64 && ipv4[1] <= 127 {
		return true
	}
	// 检查 192.0.0.0/24 (IETF 协议分配)
	if ipv4[0] == 192 && ipv4[1] == 0 && ipv4[2] == 0 {
		return true
	}
	// 检查 192.0.2.0/24 (TEST-NET-1)
	if ipv4[0] == 192 && ipv4[1] == 0 && ipv4[2] == 2 {
		return true
	}
	// 检查 198.51.100.0/24 (TEST-NET-2)
	if ipv4[0] == 198 && ipv4[1] == 51 && ipv4[2] == 100 {
		return true
	}
	// 检查 203.0.113.0/24 (TEST-NET-3)
	if ipv4[0] == 203 && ipv4[1] == 0 && ipv4[2] == 113 {
		return true
	}
	// 检查 240.0.0.0/4 (保留地址)
	if ipv4[0] >= 240 {
		return true
	}
	return false
}

func (g *GeoIPRule) Match(metadata constant.Metadata) (match bool) {
	record := constant.MMDBRecord{}
	defer func() {
		if !match {
			return
		}
		slog.Info("geoip rule match",
			"rule_country", g.country, "record_country", record.Country.ISOCode,
			"is_src", g.isSrc, "src_ip", metadata.SrcIP,
			"action", g.action)
	}()
	if g.isSrc {
		if g.country == "lan" {
			return g.isLan(metadata.SrcIP)
		}
		err := g.mmdb.Lookup(metadata.SrcIP).Decode(&record)
		if err != nil {
			slog.Error("geoip rule lookup error", "error", err, "src_ip", metadata.SrcIP)
			return false
		}
	} else {
		ips, err := resolver.Resolve(context.Background(), metadata.Host)
		if err != nil {
			slog.Error("geoip rule resolve error", "error", err, "host", metadata.Host)
			return false
		}
		if g.country == "lan" {
			return slices.ContainsFunc(ips, func(ip netip.Addr) bool {
				return g.isLan(ip)
			})
		}
		err = g.mmdb.Lookup(ips[0]).Decode(&record)
		if err != nil {
			slog.Error("geoip rule lookup error", "error", err, "host", metadata.Host)
			return false
		}
	}
	ruleCountry := strings.ToLower(g.country)
	recordCountry := strings.ToLower(record.Country.ISOCode)
	return recordCountry == ruleCountry
}

func (g *GeoIPRule) Action() constant.RuleAction {
	return g.action
}

func NewGeoIPRule(country string, action constant.RuleAction, config config.Config, isSrc bool) (*GeoIPRule, error) {
	r := GeoIPRule{
		country: country,
		action:  action,
		isSrc:   true,
	}
	mmdb, err := mmdb.LoadMMDB(config.MMDBPath.Country)
	if err != nil {
		return nil, err
	}
	r.mmdb = mmdb
	r.isSrc = isSrc
	return &r, nil
}
