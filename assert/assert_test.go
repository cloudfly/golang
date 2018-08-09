package assert

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type MockKV map[string]interface{}

func (kv MockKV) Get(key string) interface{} {
	v, _ := kv[key]
	return v
}

func TestParse(t *testing.T) {
	items, err := parse(`  A.a == B.b && 234.2342 > 234 || ("hello" == A.bcd && A.b / B.hello >= 23.24)`)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "A.a", items[0])
	assert.Equal(t, "==", items[1])
	assert.Equal(t, "B.b", items[2])
	assert.Equal(t, "&&", items[3])
	assert.Equal(t, "234.2342", items[4])
	assert.Equal(t, ">", items[5])
	assert.Equal(t, "234", items[6])
	assert.Equal(t, `||`, items[7])
	assert.Equal(t, `(`, items[8])
	assert.Equal(t, `"hello"`, items[9])
	assert.Equal(t, `A.bcd`, items[11])
	assert.Equal(t, `A.b`, items[13])
	assert.Equal(t, `/`, items[14])
	assert.Equal(t, `B.hello`, items[15])
	assert.Equal(t, `>=`, items[16])
	assert.Equal(t, `23.24`, items[17])
	assert.Equal(t, `)`, items[18])
}

func TestAssert_ExecuteNormal(t *testing.T) {
	expr, err := New("class == \"URL 监控\" && value >= 1")
	if err != nil {
		t.Fatal(err)
	}
	result, err := expr.Execute(
		MockKV(map[string]interface{}{
			"class": "URL 监控",
			"value": 234,
		}),
	)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, true, result)

	// true of false
	expr, err = New("true")
	if err != nil {
		t.Fatal(err)
	}
	result, err = expr.Execute(nil)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, true, result)

	expr, err = New("false")
	if err != nil {
		t.Fatal(err)
	}
	result, err = expr.Execute(nil)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, false, result)

	expr, err = New("abc == nil")
	if err != nil {
		t.Fatal(err)
	}
	result, err = expr.Execute(nil)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, true, result)
}

func TestAssert_ExecuteRegexp(t *testing.T) {
	expr, err := New(`s =~ "200"`)
	if err != nil {
		t.Fatal(err)
	}
	result, err := expr.Execute(
		MockKV(map[string]interface{}{
			"s": "OK 200",
		}),
	)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, true, result)

	result, err = expr.Execute(
		MockKV(map[string]interface{}{
			"s": "OK 300",
		}),
	)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, false, result)

	expr, err = New(`s !~ "200"`)
	if err != nil {
		t.Fatal(err)
	}
	result, err = expr.Execute(
		MockKV(map[string]interface{}{
			"s": "OK 200",
		}),
	)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, false, result)

}
