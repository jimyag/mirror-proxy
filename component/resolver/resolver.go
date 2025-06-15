package resolver

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"sync"

	"github.com/phuslu/fastdns"
)

var (
	DNS      string
	resolver *fastdns.Client
	once     sync.Once
)

func Resolve(ctx context.Context, host string) ([]netip.Addr, error) {
	once.Do(func() {
		resolver = &fastdns.Client{
			Addr: net.JoinHostPort(DNS, "53"),
		}
		if strings.HasPrefix(DNS, "https://") {
			endpoint, pErr := url.Parse(DNS)
			if pErr != nil {
				slog.Error("failed to parse dns", "error", pErr, "dns", DNS)
				panic(pErr)
			}
			resolver.Dialer = &fastdns.HTTPDialer{
				Endpoint: endpoint,
				Header: http.Header{
					"content-type": {"application/dns-message"},
					"user-agent":   {"mirror-proxy/1.0"},
				},
			}
		}
	})
	// 如果 host 是 IP 地址，则直接返回
	ip, err := netip.ParseAddr(strings.Split(host, ":")[0])
	if err == nil {
		return []netip.Addr{ip}, nil
	}
	req, resp := fastdns.AcquireMessage(), fastdns.AcquireMessage()
	defer fastdns.ReleaseMessage(req)
	defer fastdns.ReleaseMessage(resp)

	req.SetRequestQuestion(host, fastdns.TypeA, fastdns.ClassINET)

	err = resolver.Exchange(ctx, req, resp)
	if err != nil {
		return nil, err
	}

	ips := make([]netip.Addr, 0)
	resp.Records(func(mr fastdns.MessageRecord) bool {
		if mr.Type == fastdns.TypeA {
			v, _ := netip.AddrFromSlice(mr.Data)
			ips = append(ips, v)
		}
		return true
	})
	return ips, nil
}
