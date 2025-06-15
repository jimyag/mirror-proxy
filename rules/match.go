package rules

import (
	"github.com/jimyag/mirror-proxy/constant"
)

type MatchRule struct {
	action constant.RuleAction
}

var _ Rule = (*MatchRule)(nil)

func (m *MatchRule) Match(_ constant.Metadata) bool {
	return true
}

func (m *MatchRule) Action() constant.RuleAction {
	return m.action
}

func NewMatchRule(_ string, action constant.RuleAction) *MatchRule {
	return &MatchRule{
		action: action,
	}
}
