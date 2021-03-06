package assert

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParse(t *testing.T) {
	items, err := parse(`  A.a == B.b && 234.2342 > 234 || ("hello" == A.bcd && A.b / B.hello >= 23.24)`)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "A.a", items[0].raw)
	assert.Equal(t, "==", items[1].raw)
	assert.Equal(t, "B.b", items[2].raw)
	assert.Equal(t, "&&", items[3].raw)
	assert.Equal(t, "234.2342", items[4].raw)
	assert.Equal(t, ">", items[5].raw)
	assert.Equal(t, "234", items[6].raw)
	assert.Equal(t, `||`, items[7].raw)
	assert.Equal(t, `(`, items[8].raw)
	assert.Equal(t, `"hello"`, items[9].raw)
	assert.Equal(t, `A.bcd`, items[11].raw)
	assert.Equal(t, `A.b`, items[13].raw)
	assert.Equal(t, `/`, items[14].raw)
	assert.Equal(t, `B.hello`, items[15].raw)
	assert.Equal(t, `>=`, items[16].raw)
	assert.Equal(t, `23.24`, items[17].raw)
	assert.Equal(t, `)`, items[18].raw)
}

func TestAssert_ExecuteNormal(t *testing.T) {
	expr, err := New("class == \"URL 监控\" && value >= 1")
	if err != nil {
		t.Fatal(err)
	}
	result, err := expr.Execute(
		NewKV(map[string]interface{}{
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
	result, err = expr.Execute(NewKV(map[string]interface{}{
		"code":  200,
		"rt":    56,
		"error": "",
	}))
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, true, result)

	expr, err = New(`usage < 80`)
	if err != nil {
		t.Fatal(err)
	}
	result, err = expr.Execute(NewKV(map[string]interface{}{
		"host":  "172.16.50.50",
		"time":  1533791996370402301,
		"usage": 65.14931404914708,
	}))
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
			result, err = expr.Execute(NewKV(map[string]interface{}{
				"host":  "172.16.50.50",
				"time":  1533791996370402301,
				"usage": 65.14931404914708,
			}))
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
		NewKV(map[string]interface{}{
			"s": "OK 200",
		}),
	)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, true, result)

	result, err = expr.Execute(
		NewKV(map[string]interface{}{
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
		NewKV(map[string]interface{}{
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
		NewKV(map[string]interface{}{
			"bizType": 4,
		}),
	)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, false, result)

	result, err = expr.Execute(
		NewKV(map[string]interface{}{
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
		NewKV(map[string]interface{}{
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
			NewKV(map[string]interface{}{
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
			NewKV(map[string]interface{}{
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
			NewKV(map[string]interface{}{
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
		NewKV(map[string]interface{}{
			"value": 1000,
		}),
	)
	assert.NoError(t, err)
	assert.True(t, result)
	result, err = expr.Execute(
		NewKV(map[string]interface{}{
			"value": 100,
		}),
	)
	assert.NoError(t, err)
	assert.False(t, result)

	_, err = expr.Execute(
		NewKV(map[string]interface{}{
			"value": nil,
		}),
	)
	assert.Error(t, err)
	t.Log(err.Error())

	_, err = expr.Execute(
		NewKV(map[string]interface{}{
			"value": "gogogo",
		}),
	)
	assert.Error(t, err)
	t.Log(err.Error())

	expr, err = New(`value =~ "wskl).]"`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = expr.Execute(
		NewKV(map[string]interface{}{
			"value": 1000,
		}),
	)
	assert.Error(t, err)
	t.Log(err.Error())

}

func TestAssert_NilValue(t *testing.T) {
	expr, err := New(`value == nil || value > 1000`)
	if err != nil {
		t.Fatal(err)
	}
	ok, err := expr.Execute(
		NewKV(map[string]interface{}{
			"value": nil,
		}),
	)
	assert.NoError(t, err)
	assert.True(t, ok)

	ok, err = expr.Execute(
		NewKV(map[string]interface{}{}),
	)
	assert.NoError(t, err)
	assert.True(t, ok)

	expr, _ = New(`value != nil && value > 1000`)
	ok, err = expr.Execute(
		NewKV(map[string]interface{}{}),
	)
	assert.NoError(t, err)
	assert.False(t, ok)

	expr, _ = New(`value > 1000 || value == nil`)
	ok, err = expr.Execute(
		NewKV(map[string]interface{}{}),
	)
	// value > 1000 在前面, 先执行会出错
	assert.Error(t, err)
	t.Log(err.Error())

	expr, _ = New(`value == nil`)
	ok, err = expr.Execute(
		NewKV(map[string]interface{}{}),
	)
	assert.NoError(t, err)
	assert.True(t, ok)

	expr, _ = New(`value != nil`)
	ok, err = expr.Execute(
		NewKV(map[string]interface{}{}),
	)
	assert.NoError(t, err)
	assert.False(t, ok)

	expr, _ = New(`value == ""`)
	ok, err = expr.Execute(
		NewKV(map[string]interface{}{}),
	)
	assert.NoError(t, err)
	assert.False(t, ok)

	expr, _ = New(`!(value > 0)`)
	ok, err = expr.Execute(
		NewKV(map[string]interface{}{}),
	)
	assert.Error(t, err)
	assert.False(t, ok)
}

func TestAssert_NoValue(t *testing.T) {
	expr, _ := New(`nil == nil`)
	ok, err := expr.Execute(
		NewKV(map[string]interface{}{}),
	)
	assert.NoError(t, err)
	assert.True(t, ok)

	expr, _ = New(`123 == 123`)
	ok, err = expr.Execute(
		NewKV(map[string]interface{}{}),
	)
	assert.NoError(t, err)
	assert.True(t, ok)

	expr, _ = New(`"123" == "123"`)
	ok, err = expr.Execute(
		NewKV(map[string]interface{}{}),
	)
	assert.NoError(t, err)
	assert.True(t, ok)
}

func TestEqual(t *testing.T) {
	assert.True(t, Equal("", ""))
	assert.True(t, Equal("a==2", "a == 2"))
	assert.False(t, Equal("a=2", "a == 2"))
}

func BenchmarkAssert(b *testing.B) {
	expr, err := New("a==b")
	if err != nil {
		b.Fatal(err)
	}
	expri, err := New("ia>ib")
	if err != nil {
		b.Fatal(err)
	}
	kv := NewKV(map[string]interface{}{"a": "Hello", "b": "World", "ia": 12, "ib": 44})
	for i := 0; i < b.N; i++ {
		expr.Execute(kv)
		expri.Execute(kv)
	}
}
