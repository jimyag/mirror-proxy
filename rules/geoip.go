package rules

import (
	"log/slog"
	"net/netip"

	"github.com/oschwald/maxminddb-golang/v2"

	"github.com/jimyag/mirror-proxy/component/mmdb"
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

func (g *GeoIPRule) Match(metadata constant.Metadata) (res bool) {
	defer func() {
		action := constant.RuleActionDeny
		if res {
			action = constant.RuleActionAllow
		}
		slog.Info("geoip rule match",
			"country", g.country, "is_src", g.isSrc,
			"src_ip", metadata.SrcIP, "action", g.action, "result", action)
	}()
	record := constant.MMDBRecord{}
	if g.isSrc {
		err := g.mmdb.Lookup(metadata.SrcIP).Decode(&record)
		if err != nil {
			return false
		}
	} else {
		hostIP, err := netip.ParseAddr(metadata.Host)
		if err != nil {
			return false
		}
		err = g.mmdb.Lookup(hostIP).Decode(&record)
		if err != nil {
			return false
		}
	}
	if record.Country.ISOCode == g.country {
		return true
	}
	return false
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
