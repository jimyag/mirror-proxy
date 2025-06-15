package rules

import (
	"fmt"
	"strings"

	"github.com/jimyag/mirror-proxy/config"
	"github.com/jimyag/mirror-proxy/constant"
)

type Rule interface {
	// Match 检查请求是否匹配规则
	Match(metadata constant.Metadata) bool
	// Action 返回规则的行为
	Action() constant.RuleAction
}

func ParseRule(ruleStr string, cfg config.Config) (Rule, error) {
	rule := strings.Split(ruleStr, ",")
	if len(rule) < 2 {
		return nil, fmt.Errorf("invalid rule: %s", ruleStr)
	}
	ty := rule[0]
	action := constant.RuleAction(rule[len(rule)-1])
	payload := strings.Join(rule[1:len(rule)-1], ",")
	if !action.Validate() {
		return nil, fmt.Errorf("invalid rule action: %s", action)
	}
	switch ty {
	case "domain":
		return NewDomainRule(payload, action)
	case "match":
		return NewMatchRule(payload, action), nil
	case "src-ip":
		return NewGeoIPRule(payload, action, cfg, true)
	case "dst-ip":
		return NewGeoIPRule(payload, action, cfg, false)
	default:
		return nil, fmt.Errorf("invalid rule type: %s", ty)
	}
}
