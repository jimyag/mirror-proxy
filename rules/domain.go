package rules

import (
	"log/slog"
	"net/url"

	"github.com/jimyag/mirror-proxy/constant"
)

var _ Rule = (*DomainRule)(nil)

type DomainRule struct {
	// domain,<payload>,<action>
	// domain,https://github.com,allow
	// domain,raw.githubusercontent.com,allow
	host       string
	action     constant.RuleAction
	allowHTTP  bool
	allowHTTPS bool
}

// NewDomainRule 创建域名规则
func NewDomainRule(domain string, action constant.RuleAction) (*DomainRule, error) {
	u, err := url.Parse(domain)
	if err != nil {
		return nil, err
	}

	r := DomainRule{
		host:       u.Host,
		action:     action,
		allowHTTP:  u.Scheme == "http",
		allowHTTPS: u.Scheme == "https",
	}

	if u.Scheme == "" {
		r.allowHTTP = true
		r.allowHTTPS = true
	}
	return &r, nil
}

// Match 检查域名是否匹配
func (r *DomainRule) Match(metadata constant.Metadata) (res bool) {
	defer func() {
		action := constant.RuleActionDeny
		if res {
			action = constant.RuleActionAllow
		}
		slog.Info("domain rule match",
			"host", metadata.Host, "protocol", metadata.Protocol,
			"action", r.action, "result", action)
	}()
	if metadata.Host != r.host {
		return false
	}

	// 如果是 http 协议，但是不允许 http 协议，则不匹配
	if metadata.Protocol == "http" && !r.allowHTTP {
		return false
	}

	// 如果是 https 协议，但是不允许 https 协议，则不匹配
	if metadata.Protocol == "https" && !r.allowHTTPS {
		return false
	}

	return true
}

// Action 返回规则行为
func (r *DomainRule) Action() constant.RuleAction {
	return r.action
}
