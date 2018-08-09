package assert

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

const (
	Nil = iota
	Boolean
	Number
	String
)

type Value struct {
	val   interface{}
	vType uint8
}

func NewValue(v interface{}) Value {
	res := Value{
		val:   v,
		vType: Nil,
	}
	if v == nil {
		return res
	}
	switch r := v.(type) {
	case bool:
		res.vType = Boolean
	case []byte:
		res.val = string(r)
		res.vType = String
	case string:
		res.vType = String
	case float64:
		res.vType = Number
	case float32:
		res.val = float64(r)
		res.vType = Number
	case int:
		res.val = float64(r)
		res.vType = Number
	case int8:
		res.val = float64(r)
		res.vType = Number
	case int16:
		res.val = float64(r)
		res.vType = Number
	case int32:
		res.val = float64(r)
		res.vType = Number
	case int64:
		res.val = float64(r)
		res.vType = Number
	case uint:
		res.val = float64(r)
		res.vType = Number
	case uint8:
		res.val = float64(r)
		res.vType = Number
	case uint16:
		res.val = float64(r)
		res.vType = Number
	case uint32:
		res.val = float64(r)
		res.vType = Number
	case uint64:
		res.val = float64(r)
		res.vType = Number
	}
	return res
}

func (v Value) Float() (float64, error) {
	switch v.vType {
	case Number:
		return v.val.(float64), nil
	case String:
		return strconv.ParseFloat(fmt.Sprintf("%v", v.val), 64)
	}
	return 0, fmt.Errorf("value '%v' not a number", v.val)
}

func (v Value) String() string {
	if v.vType == String {
		return v.val.(string)
	}
	return fmt.Sprintf("%v", v.val)
}

func (v Value) Boolean() bool {
	if v.vType == Boolean {
		return v.val.(bool)
	}
	s := strings.ToLower(fmt.Sprintf("%v", v.val))
	return s != "" && s != "false"
}

func (v Value) Not() Value {
	return Value{
		val:   !v.Boolean(),
		vType: Boolean,
	}
}

func (v Value) And(v2 Value) Value {
	return Value{
		val:   v.Boolean() && v2.Boolean(),
		vType: Boolean,
	}
}

func (v Value) Or(v2 Value) Value {
	return Value{
		val:   v.Boolean() || v2.Boolean(),
		vType: Boolean,
	}
}

func (v Value) E(v2 Value) Value {
	return Value{
		val:   v.String() == v2.String(),
		vType: Boolean,
	}
}

func (v Value) RE(v2 Value) Value {
	exp, err := regexp.Compile(v2.String())
	if err != nil {
		return Value{
			val:   false,
			vType: Boolean,
		}
	}

	return Value{
		val:   exp.MatchString(v.String()),
		vType: Boolean,
	}
}

func (v Value) NRE(v2 Value) Value {
	exp, err := regexp.Compile(v2.String())
	if err != nil {
		return Value{
			val:   false,
			vType: Boolean,
		}
	}

	return Value{
		val:   !exp.MatchString(v.String()),
		vType: Boolean,
	}
}

func (v Value) NE(v2 Value) Value {
	return Value{
		val:   v.String() != v2.String(),
		vType: Boolean,
	}
}

func (v Value) GT(v2 Value) Value {
	left, err := v.Float()
	if err != nil {
		return Value{val: false, vType: Boolean}
	}
	right, err := v2.Float()
	if err != nil {
		return Value{val: false, vType: Boolean}
	}
	return Value{
		val:   left > right,
		vType: Boolean,
	}
}

func (v Value) GTE(v2 Value) Value {
	left, err := v.Float()
	if err != nil {
		return Value{val: false, vType: Boolean}
	}
	right, err := v2.Float()
	if err != nil {
		return Value{val: false, vType: Boolean}
	}
	return Value{
		val:   left >= right,
		vType: Boolean,
	}
}

func (v Value) LT(v2 Value) Value {
	left, err := v.Float()
	if err != nil {
		return Value{val: false, vType: Boolean}
	}
	right, err := v2.Float()
	if err != nil {
		return Value{val: false, vType: Boolean}
	}
	return Value{
		val:   left < right,
		vType: Boolean,
	}
}

func (v Value) LTE(v2 Value) Value {
	left, err := v.Float()
	if err != nil {
		return Value{val: false, vType: Boolean}
	}
	right, err := v2.Float()
	if err != nil {
		return Value{val: false, vType: Boolean}
	}
	return Value{
		val:   left <= right,
		vType: Boolean,
	}
}

func (v Value) Add(v2 Value) Value {
	f, err := v.Float()
	if err != nil {
		return Value{val: v.String() + v2.String(), vType: String}
	}
	f2, err := v2.Float()
	if err != nil {
		return Value{val: v.String() + v2.String(), vType: String}
	}
	return Value{
		val:   f + f2,
		vType: Number,
	}
}

func (v Value) Sub(v2 Value) (Value, error) {
	f, err := v.Float()
	if err != nil {
		return Value{}, err
	}
	f2, err := v2.Float()
	if err != nil {
		return Value{}, err
	}
	return Value{
		val:   f - f2,
		vType: Number,
	}, nil
}

func (v Value) Multi(v2 Value) (Value, error) {
	f, err := v.Float()
	if err != nil {
		return Value{}, err
	}
	f2, err := v2.Float()
	if err != nil {
		return Value{}, err
	}
	return Value{
		val:   f * f2,
		vType: Number,
	}, nil
}

func (v Value) Div(v2 Value) (Value, error) {
	f, err := v.Float()
	if err != nil {
		return Value{}, err
	}
	f2, err := v2.Float()
	if err != nil {
		return Value{}, err
	}
	return Value{
		val:   f / f2,
		vType: Number,
	}, nil
}

func (v Value) Mod(v2 Value) (Value, error) {
	f, err := v.Float()
	if err != nil {
		return Value{}, err
	}
	f2, err := v2.Float()
	if err != nil {
		return Value{}, err
	}
	return Value{
		val:   float64(int(f) % int(f2)),
		vType: Number,
	}, nil
}
