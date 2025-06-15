package constant

type RuleAction string

const (
	RuleActionAllow RuleAction = "allow"
	RuleActionDeny  RuleAction = "deny"
)

func (a RuleAction) Validate() bool {
	switch a {
	case RuleActionAllow, RuleActionDeny:
		return true
	default:
		return false
	}
}

func (a RuleAction) String() string {
	return string(a)
}
