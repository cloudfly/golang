package template

import (
	"testing"
)

var (
	ctx       Context
	dataTable = map[string]interface{}{
		"name": "jack",
		"age":  28,
		"parents": map[string]string{
			"father": "tom",
			"mother": "lucy",
		},
		"data": map[string]interface{}{
			"nil": nil,
		},
		"incr":     func(n int) int { n++; return n },
		"istrue":   true,
		"isfalse":  false,
		"emptystr": "",
		"zero":     0,
		"nil":      nil,
	}
	stringTable = map[string]string{
		"name is jack":   "name is {{ .name }}",
		"age is 28":      "age is {{ .age }}",
		"mother is lucy": "mother is {{ .parents.mother }}",
		"age is 29":      "age is {{ .incr .age }}",
		" true":          " {{ eq 28 .age }}",
	}
	boolTable = []struct {
		answer   bool
		template string
	}{
		{true, "{{ .istrue }}"},
		{false, "{{ .isfalse}}"},
		{false, "{{ .emptystr | bool }}"},
		{true, "{{ .name | bool }}"},
		{false, "{{ .zero | bool }}"},
		{false, "{{ .data.nil | bool }}"},
		{true, "{{ and true .name }}"},
		{false, "{{ and .emptystr .name }}"},
		{true, "{{ not .emptystr }}"},
		{true, "{{ or .emptystr .name }}"},
		{false, "{{ and .name .data.nil }}"},
		{true, "{{ eq .name .name }}"},
		{true, "{{ ne .parents.father .parents.mother }}"},
		{true, "{{ gt .age 10 }}"},
		{false, "{{ lt .age 10 }}"},
		{true, "{{ ge .age 28 }}"},
		{true, "{{ le .age 28 }}"},
	}
)

type Context struct {
	data map[string]interface{}
}

func (c Context) Value(key string) interface{} {
	if d, ok := c.data[key]; ok {
		return d
	}
	return nil
}

func init() {
	ctx = Context{
		data: dataTable,
	}
}

func TestExecute(t *testing.T) {
	for answer, setting := range stringTable {
		result, err := Parse(setting, ctx)
		if err != nil {
			t.Error(err)
		}
		t.Logf("%-40s  \"%v\" == \"%s\"", setting, result, answer)
		if result != answer {
			t.Error("uncorrect parse result")
		}
	}
	for _, item := range boolTable {
		result, err := Parse(item.template, ctx)
		if err != nil {
			t.Error(err)
		}
		t.Logf("%-40s  \"%v\" == \"%t\"", item.template, result, item.answer)
		if result != item.answer {
			t.Error("uncorrect parse result")
		}
	}
}
