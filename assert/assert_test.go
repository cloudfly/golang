package assert

import (
	"sync"
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

	expr, err = New(`code >= 200 && code < 300 && rt < 10000 && error == ''`)
	if err != nil {
		t.Fatal(err)
	}
	result, err = expr.Execute(MockKV{
		"code":  200,
		"rt":    56,
		"error": "",
	})
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, true, result)

	expr, err = New(`usage < 80`)
	if err != nil {
		t.Fatal(err)
	}
	result, err = expr.Execute(MockKV{
		"host":  "172.16.50.50",
		"time":  1533791996370402301,
		"usage": 65.14931404914708,
	})
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, true, result)

	// test in parallel
	wait := sync.WaitGroup{}
	wait.Add(20)
	for i := 0; i < 20; i++ {
		go func() {
			defer wait.Done()
			result, err = expr.Execute(MockKV{
				"host":  "172.16.50.50",
				"time":  1533791996370402301,
				"usage": 65.14931404914708,
			})
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, true, result)
		}()
	}
	wait.Wait()
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

	expr, err = New(`bizType >= -10 && bizType <= -1`)
	if err != nil {
		t.Fatal(err)
	}
	result, err = expr.Execute(
		MockKV(map[string]interface{}{
			"bizType": 4,
		}),
	)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, false, result)

	result, err = expr.Execute(
		MockKV(map[string]interface{}{
			"bizType": -5,
		}),
	)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, true, result)

	expr, err = New(`rx_mbps < 500 || rx_mbps > 20000 || rx_mbps == nil `)
	if err != nil {
		t.Fatal(err)
	}
	result, err = expr.Execute(
		MockKV(map[string]interface{}{
			"rx_mbps": 4000,
		}),
	)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, false, result)

}

func TestAssert_ExecuteMatch(t *testing.T) {
	expr, err := New(`host = '*.50.50'`)
	if err != nil {
		t.Fatal(err)
	}

	for _, item := range []string{
		"10.1.50.50",
		".50.50",
		"abc.50.50",
	} {
		result, err := expr.Execute(
			MockKV(map[string]interface{}{
				"host": item,
			}),
		)
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, true, result)
	}

	for _, item := range []string{
		"10.150.50",
		"50.50",
		"abc50.50",
		"abc.50.50.10",
	} {
		result, err := expr.Execute(
			MockKV(map[string]interface{}{
				"host": item,
			}),
		)
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, false, result)
	}

	expr, err = New(`host = "*50.50*"`)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range []string{
		"10.150.50",
		"50.50",
		"abc50.50",
		"abc.50.50.10",
	} {
		result, err := expr.Execute(
			MockKV(map[string]interface{}{
				"host": item,
			}),
		)
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, true, result)
	}

}

func TestAssert_Error(t *testing.T) {
	expr, err := New(`value > 200`)
	if err != nil {
		t.Fatal(err)
	}

	result, err := expr.Execute(
		MockKV(map[string]interface{}{
			"value": 1000,
		}),
	)
	assert.NoError(t, err)
	assert.True(t, result)
	result, err = expr.Execute(
		MockKV(map[string]interface{}{
			"value": 100,
		}),
	)
	assert.NoError(t, err)
	assert.False(t, result)

	_, err = expr.Execute(
		MockKV(map[string]interface{}{
			"value": nil,
		}),
	)
	assert.Error(t, err)
	assert.Equal(t, "variable is a nil value", err.Error())

	_, err = expr.Execute(
		MockKV(map[string]interface{}{
			"value": "gogogo",
		}),
	)
	assert.Error(t, err)
	assert.Equal(t, "variable is not a number", err.Error())

	expr, err = New(`value =~ "wskl).]"`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = expr.Execute(
		MockKV(map[string]interface{}{
			"value": 1000,
		}),
	)
	assert.Error(t, err)
	assert.Equal(t, "invalid regexp", err.Error())

	expr, err = New(`value == nil`)
	if err != nil {
		t.Fatal(err)
	}
	ok, err := expr.Execute(
		MockKV(map[string]interface{}{
			"value": nil,
		}),
	)
	assert.NoError(t, err)
	assert.True(t, ok)

	ok, err = expr.Execute(
		MockKV(map[string]interface{}{}),
	)
	assert.NoError(t, err)
	assert.True(t, ok)
}
