package filterpolicy

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseFilterPolicy(t *testing.T) {
	_, err := ParseFilterPolicy("*/2")
	if err != nil {
		t.Error(err)
	}

	_, err = ParseFilterPolicy("-")
	assert.NoError(t, err)

	_, err = ParseFilterPolicy("abc")
	if err == nil {
		t.Error("policy 'abc' should be return error")
	}

	_, err = NewFilterPolicyItem("*/abc")
	if err == nil {
		t.Errorf("*/abc should be a unvalid filter item")
	}

	_, err = NewFilterPolicyItem("2-b")
	if err == nil {
		t.Errorf("'2-b' should be a unvalid filter item")
	}
	_, err = NewFilterPolicyItem("a-3")
	if err == nil {
		t.Errorf("'a-3' should be a unvalid filter item")
	}
}

func TestFilterPolicyItem_Pass(t *testing.T) {
	_, err := NewFilterPolicyItem("")
	if err == nil {
		t.Fatal(err)
	}
	item, err := NewFilterPolicyItem("23")
	if err != nil {
		t.Fatal(err)
	}
	if !item.Pass(23) {
		t.Error("23 should be passed")
	}
	if item.Pass(3) {
		t.Error("3 should not be passed")
	}

	item, err = NewFilterPolicyItem("*/3")
	if err != nil {
		t.Fatal(err)
	}
	if !item.Pass(3) || !item.Pass(6) || !item.Pass(33) {
		t.Error("3,6,33 should be passed")
	}
	if item.Pass(22) || item.Pass(0) || item.Pass(1) {
		t.Error("22,0,1 should not be passed")
	}

	item, err = NewFilterPolicyItem("1-4")
	if err != nil {
		t.Fatal(err)
	}
	if !(item.Pass(1) && item.Pass(2) && item.Pass(3) && item.Pass(4)) {
		t.Error("1-4 should be passed")
	}
	if item.Pass(5) || item.Pass(0) {
		t.Error("0, 5 should not be passed")
	}
}

func TestFilterPolicy_Pass(t *testing.T) {
	policy, err := ParseFilterPolicy("")
	if err != nil {
		t.Error(err)
	}
	if !policy.Pass(1) || !policy.Pass(1324) {
		t.Error("1, 1324 should be passed")
	}
	policy, err = ParseFilterPolicy("1,2,3,25,100")
	if err != nil {
		t.Error(err)
	}
	if !policy.Pass(1) || !policy.Pass(3) || !policy.Pass(25) || !policy.Pass(100) {
		t.Error("1,3,5,100 should be passed")
	}
	if policy.Pass(0) || policy.Pass(4) || policy.Pass(12) {
		t.Error("0,4,12 should not be passed")
	}
	policy, err = ParseFilterPolicy("1")

	if !policy.Pass(1) {
		t.Error("1 should be passed")
	}
	if policy.Pass(2) || policy.Pass(3) || policy.Pass(4) {
		t.Error("2, 3, 4 should not be passed")
	}

	policy, err = ParseFilterPolicy("-")
	assert.NoError(t, err)
	if policy.Pass(1) || policy.Pass(10) || policy.Pass(100) {
		t.Error("1, 10, 100 should not be passed")
	}
}

func TestFilterPolicy_PassedBefore(t *testing.T) {
	policy, err := ParseFilterPolicy("*/5")
	if err != nil {
		t.Error(err)
	}

	assert.Equal(t, false, policy.PassedBefore(3))
	assert.Equal(t, true, policy.PassedBefore(6))
	assert.Equal(t, true, policy.PassedBefore(5))
}
