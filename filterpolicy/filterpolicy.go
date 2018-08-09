package filterpolicy

import (
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

// Kinds of policy
const (
	policyKindSingle = iota
	policyKindScope
	policyKindInterval
	policyKindNever
)

// FilterPolicy is a real policy executor
type FilterPolicy struct {
	spec  string
	items []*FilterPolicyItem
}

// ParseFilterPolicy create a new FilterPolicy from a given specification
// eg. 1,2,3,25,100 will pass 1,2,3,25 and 100. others will not
func ParseFilterPolicy(spec string) (*FilterPolicy, error) {
	policy := &FilterPolicy{
		spec:  spec,
		items: make([]*FilterPolicyItem, 0, 10),
	}
	fields := strings.Split(spec, ",")
	for _, field := range fields {
		f := strings.TrimSpace(field)
		if f == "" {
			continue
		}
		item, err := NewFilterPolicyItem(f)
		if err != nil {
			return nil, errors.Wrapf(err, "unvalid spec '%s'", field)
		}
		policy.items = append(policy.items, item)
	}
	return policy, nil
}

// Pass check if a given ingeter can pass the policy
func (policy *FilterPolicy) Pass(i int) bool {
	if policy.spec == "" {
		return true
	}
	for _, item := range policy.items {
		if item.Pass(i) {
			return true
		}
	}
	return false
}

// PassedBefore check if the policy pass number which smaller(or equal) than repeat
func (policy *FilterPolicy) PassedBefore(repeat int) bool {
	for i := 1; i <= repeat; i++ {
		if policy.Pass(i) {
			return true
		}
	}
	return false
}

// FilterPolicyItem is a policy item, a part of FilterPolicyItem
type FilterPolicyItem struct {
	spec     string
	kind     int
	single   int
	scope    [2]int
	interval int
}

// NewFilterPolicyItem create a new policy item from a specification
// <integer>: represents exact match
// */<integer>: represents the number should be divisible by <ingeger>
// <max-integer>-<max-ingeter>: represents a number range, only numbers in this range can be passed
// eg. 34 or */3 or 3-12
func NewFilterPolicyItem(spec string) (*FilterPolicyItem, error) {
	if spec == "" {
		return nil, errors.New("empty specification")
	}

	if spec == "-" {
		return &FilterPolicyItem{
			spec: spec,
			kind: policyKindNever,
		}, nil
	}

	if strings.HasPrefix(spec, "*/") && len(spec) >= 3 {
		interval := spec[2:]
		i, err := strconv.Atoi(interval)
		if err != nil {
			return nil, errors.Wrapf(err, "unvalid integer %s", interval)
		}
		return &FilterPolicyItem{
			spec:     spec,
			kind:     policyKindInterval,
			interval: i,
		}, nil
	}

	if index := strings.IndexByte(spec, '-'); index != -1 && len(spec) > index+1 {
		mins, maxs := spec[:index], spec[index+1:]
		min, err := strconv.Atoi(mins)
		if err != nil {
			return nil, errors.Wrapf(err, "unvalid integer %s", mins)
		}
		max, err := strconv.Atoi(maxs)
		if err != nil {
			return nil, errors.Wrapf(err, "unvalid integer %s", maxs)
		}
		return &FilterPolicyItem{
			spec:  spec,
			kind:  policyKindScope,
			scope: [2]int{min, max},
		}, nil
	}

	i, err := strconv.Atoi(spec)
	if err != nil {
		return nil, errors.Wrapf(err, "unvalid integer %s", spec)
	}
	return &FilterPolicyItem{
		spec:   spec,
		kind:   policyKindSingle,
		single: i,
	}, nil
}

// Pass check if the given integer can pass the policy
func (item *FilterPolicyItem) Pass(i int) bool {
	if i <= 0 {
		return false
	}
	switch item.kind {
	case policyKindSingle:
		return item.single == i
	case policyKindScope:
		return item.scope[0] <= i && item.scope[1] >= i
	case policyKindInterval:
		return i%item.interval == 0
	case policyKindNever:
		return false
	}
	return true
}
