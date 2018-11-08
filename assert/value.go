package assert

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/cloudfly/golang/tools"
	"github.com/pkg/errors"
)

const (
	Nil = iota
	Boolean
	Number
	String
	Error
)

type Value struct {
	name  string
	val   interface{}
	vType uint8
}

func NewValue(name string, v interface{}) Value {
	res := Value{
		name:  name,
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
	default:
		s := fmt.Sprintf("%v", v)
		f, err := strconv.ParseFloat(s, 64)
		if err == nil {
			res.val = f
			res.vType = Number
			break
		}
		b, err := strconv.ParseBool(s)
		if err == nil {
			res.val = b
			res.vType = Boolean
			break
		}
		res.val = s
		res.vType = String
	}
	return res
}

func (v Value) Float() (float64, error) {
	switch v.vType {
	case Number:
		return v.val.(float64), nil
	case String:
		f, err := strconv.ParseFloat(fmt.Sprintf("%v", v.val), 64)
		if err != nil {
			return 0, err
		}
		return f, nil
	case Nil:
		if v.name == "" {
			return 0, errors.New("can not convert nil value to number")
		} else {
			return 0, fmt.Errorf("variable '%s' is nil, can not convert to number", v.name)
		}
	case Error:
		return 0, fmt.Errorf("%v", v.val)
	}
	if v.name == "" {
		return 0, errors.New("unknown value type")
	}
	return 0, fmt.Errorf("unknow value type of variable '%s'", v.name)
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
	} else if v.vType == Error || v.vType == Nil {
		return false
	}
	s := strings.ToLower(fmt.Sprintf("%v", v.val))
	return s != "" && s != "false"
}

func (v Value) Error() error {
	if v.vType == Error {
		return fmt.Errorf("%v", v.val)
	}
	return nil
}

func (v Value) Not() Value {
	return Value{
		val:   !v.Boolean(),
		vType: Boolean,
	}
}

func (v Value) And(v2 Value) Value {
	if v.vType == Error {
		return v
	}
	if !v.Boolean() {
		return v
	}
	return v2
}

func (v Value) Or(v2 Value) Value {
	if v.vType == Error {
		return v
	}
	if v.Boolean() {
		return v
	}
	return v2
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
			val:   err.Error(),
			vType: Error,
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
			val:   err.Error(),
			vType: Error,
		}
	}

	return Value{
		val:   !exp.MatchString(v.String()),
		vType: Boolean,
	}
}

func (v Value) NE(v2 Value) Value {
	return v.E(v2).Not()
}

func (v Value) GT(v2 Value) Value {
	left, err := v.Float()
	if err != nil {
		return Value{
			val:   err.Error(),
			vType: Error,
		}
	}
	right, err := v2.Float()
	if err != nil {
		return Value{
			val:   err.Error(),
			vType: Error,
		}
	}
	return Value{
		val:   left > right,
		vType: Boolean,
	}
}

func (v Value) GTE(v2 Value) Value {
	left, err := v.Float()
	if err != nil {
		return Value{
			val:   err.Error(),
			vType: Error,
		}
	}
	right, err := v2.Float()
	if err != nil {
		return Value{
			val:   err.Error(),
			vType: Error,
		}
	}
	return Value{
		val:   left >= right,
		vType: Boolean,
	}
}

func (v Value) LT(v2 Value) Value {
	left, err := v.Float()
	if err != nil {
		return Value{
			val:   err.Error(),
			vType: Error,
		}
	}
	right, err := v2.Float()
	if err != nil {
		return Value{
			val:   err.Error(),
			vType: Error,
		}
	}
	return Value{
		val:   left < right,
		vType: Boolean,
	}
}

func (v Value) LTE(v2 Value) Value {
	left, err := v.Float()
	if err != nil {
		return Value{
			val:   err.Error(),
			vType: Error,
		}
	}
	right, err := v2.Float()
	if err != nil {
		return Value{
			val:   err.Error(),
			vType: Error,
		}
	}
	return Value{
		val:   left <= right,
		vType: Boolean,
	}
}

func (v Value) MATCH(v2 Value) Value {
	return Value{
		val:   tools.SimpleMatch(v2.String(), v.String()),
		vType: Boolean,
	}
}

func (v Value) Add(v2 Value) Value {
	f, err := v.Float()
	if err != nil {
		return Value{
			val:   v.String() + v2.String(),
			vType: String,
		}
	}
	f2, err := v2.Float()
	if err != nil {
		return Value{
			val:   v.String() + v2.String(),
			vType: String,
		}
	}
	return Value{
		val:   f + f2,
		vType: Number,
	}
}

func (v Value) Sub(v2 Value) Value {
	f, err := v.Float()
	if err != nil {
		return Value{
			val:   err.Error(),
			vType: Error,
		}
	}
	f2, err := v2.Float()
	if err != nil {
		return Value{
			val:   err.Error(),
			vType: Error,
		}
	}
	return Value{
		val:   f - f2,
		vType: Number,
	}
}

func (v Value) Multi(v2 Value) Value {
	f, err := v.Float()
	if err != nil {
		return Value{
			val:   err.Error(),
			vType: Error,
		}
	}
	f2, err := v2.Float()
	if err != nil {
		return Value{
			val:   err.Error(),
			vType: Error,
		}
	}
	return Value{
		val:   f * f2,
		vType: Number,
	}
}

func (v Value) Div(v2 Value) Value {
	f, err := v.Float()
	if err != nil {
		return Value{
			val:   err.Error(),
			vType: Error,
		}
	}
	f2, err := v2.Float()
	if err != nil {
		return Value{
			val:   err.Error(),
			vType: Error,
		}
	}
	return Value{
		val:   f / f2,
		vType: Number,
	}
}

func (v Value) Mod(v2 Value) Value {
	f, err := v.Float()
	if err != nil {
		return Value{
			val:   err.Error(),
			vType: Error,
		}
	}
	f2, err := v2.Float()
	if err != nil {
		return Value{
			val:   err.Error(),
			vType: Error,
		}
	}
	return Value{
		val:   float64(int(f) % int(f2)),
		vType: Number,
	}
}
