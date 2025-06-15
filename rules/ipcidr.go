package rules

import (
	"context"
	"log/slog"
	"net/netip"
	"slices"

	"github.com/jimyag/mirror-proxy/component/resolver"
	"github.com/jimyag/mirror-proxy/constant"
)

func NewIpCIDRRule(ipcidr string, action constant.RuleAction, isSrc bool, dns string) (*IpCIDRRule, error) {
	ipnet, err := netip.ParsePrefix(ipcidr)
	if err != nil {
		return nil, err
	}
	r := IpCIDRRule{
		ipnet:  ipnet,
		action: action,
		isSrc:  isSrc,
	}
	return &r, nil
}

type IpCIDRRule struct {
	ipnet  netip.Prefix
	action constant.RuleAction
	isSrc  bool
}

func (i *IpCIDRRule) Match(metadata constant.Metadata) (match bool) {
	ips := make([]netip.Addr, 0)
	defer func() {
		if !match {
			return
		}
		slog.Info("ipcidr rule match",
			"rule_ipcidr", i.ipnet.String(), "record_ipcidr", ips,
			"is_src", i.isSrc, "action", i.action,
			"src_ip", metadata.SrcIP, "host", metadata.Host)
	}()
	if i.isSrc {
		ips = append(ips, metadata.SrcIP)
	} else {
		var err error
		ips, err = resolver.Resolve(context.Background(), metadata.Host)
		if err != nil {
			slog.Error("ipcidr rule resolve error", "error", err, "host", metadata.Host)
			return false
		}
	}
	return slices.ContainsFunc(ips, func(ip netip.Addr) bool {
		return i.ipnet.Contains(ip)
	})
}

func (i *IpCIDRRule) Action() constant.RuleAction {
	return i.action
}
