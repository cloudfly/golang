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
	NoError = iota
	ErrNotNumber
	ErrNilValue
	ErrInvalidRegexp
)

func code2err(code int) error {
	switch code {
	case NoError:
		return nil
	case ErrNotNumber:
		return errors.New("variable is not a number")
	case ErrNilValue:
		return errors.New("variable is a nil value")
	case ErrInvalidRegexp:
		return errors.New("invalid regexp")
	}
	return nil
}

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

func (v Value) Float() (float64, int) {
	switch v.vType {
	case Number:
		return v.val.(float64), NoError
	case String:
		f, err := strconv.ParseFloat(fmt.Sprintf("%v", v.val), 64)
		if err != nil {
			return 0, ErrNotNumber
		}
		return f, NoError
	case Nil:
		return 0, ErrNilValue
	}
	return 0, ErrNotNumber
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

func (v Value) Not() (Value, int) {
	return Value{
		val:   !v.Boolean(),
		vType: Boolean,
	}, NoError
}

func (v Value) And(v2 Value) (Value, int) {
	return Value{
		val:   v.Boolean() && v2.Boolean(),
		vType: Boolean,
	}, NoError
}

func (v Value) Or(v2 Value) (Value, int) {
	return Value{
		val:   v.Boolean() || v2.Boolean(),
		vType: Boolean,
	}, NoError
}

func (v Value) E(v2 Value) (Value, int) {
	return Value{
		val:   v.String() == v2.String(),
		vType: Boolean,
	}, NoError
}

func (v Value) RE(v2 Value) (Value, int) {
	exp, err := regexp.Compile(v2.String())
	if err != nil {
		return Value{
			val:   false,
			vType: Boolean,
		}, ErrInvalidRegexp
	}
	return Value{
		val:   exp.MatchString(v.String()),
		vType: Boolean,
	}, NoError
}

func (v Value) NRE(v2 Value) (Value, int) {
	exp, err := regexp.Compile(v2.String())
	if err != nil {
		return Value{
			val:   false,
			vType: Boolean,
		}, ErrInvalidRegexp
	}

	return Value{
		val:   !exp.MatchString(v.String()),
		vType: Boolean,
	}, NoError
}

func (v Value) NE(v2 Value) (Value, int) {
	return Value{
		val:   v.String() != v2.String(),
		vType: Boolean,
	}, NoError
}

func (v Value) GT(v2 Value) (Value, int) {
	left, err := v.Float()
	if err != NoError {
		return Value{val: false, vType: Boolean}, err
	}
	right, err := v2.Float()
	if err != NoError {
		return Value{val: false, vType: Boolean}, err
	}
	return Value{
		val:   left > right,
		vType: Boolean,
	}, NoError
}

func (v Value) GTE(v2 Value) (Value, int) {
	left, err := v.Float()
	if err != NoError {
		return Value{val: false, vType: Boolean}, err
	}
	right, err := v2.Float()
	if err != NoError {
		return Value{val: false, vType: Boolean}, err
	}
	return Value{
		val:   left >= right,
		vType: Boolean,
	}, NoError
}

func (v Value) LT(v2 Value) (Value, int) {
	left, err := v.Float()
	if err != NoError {
		return Value{val: false, vType: Boolean}, err
	}
	right, err := v2.Float()
	if err != NoError {
		return Value{val: false, vType: Boolean}, err
	}
	return Value{
		val:   left < right,
		vType: Boolean,
	}, NoError
}

func (v Value) LTE(v2 Value) (Value, int) {
	left, err := v.Float()
	if err != NoError {
		return Value{val: false, vType: Boolean}, err
	}
	right, err := v2.Float()
	if err != NoError {
		return Value{val: false, vType: Boolean}, err
	}
	return Value{
		val:   left <= right,
		vType: Boolean,
	}, NoError
}

func (v Value) MATCH(v2 Value) (Value, int) {
	return Value{
		val:   tools.SimpleMatch(v2.String(), v.String()),
		vType: Boolean,
	}, NoError
}

func (v Value) Add(v2 Value) (Value, int) {
	f, err := v.Float()
	if err != NoError {
		return Value{val: v.String() + v2.String(), vType: String}, err
	}
	f2, err := v2.Float()
	if err != NoError {
		return Value{val: v.String() + v2.String(), vType: String}, err
	}
	return Value{
		val:   f + f2,
		vType: Number,
	}, NoError
}

func (v Value) Sub(v2 Value) (Value, int) {
	f, err := v.Float()
	if err != NoError {
		return Value{}, err
	}
	f2, err := v2.Float()
	if err != NoError {
		return Value{}, err
	}
	return Value{
		val:   f - f2,
		vType: Number,
	}, NoError
}

func (v Value) Multi(v2 Value) (Value, int) {
	f, err := v.Float()
	if err != NoError {
		return Value{}, err
	}
	f2, err := v2.Float()
	if err != NoError {
		return Value{}, err
	}
	return Value{
		val:   f * f2,
		vType: Number,
	}, NoError
}

func (v Value) Div(v2 Value) (Value, int) {
	f, err := v.Float()
	if err != NoError {
		return Value{}, err
	}
	f2, err := v2.Float()
	if err != NoError {
		return Value{}, err
	}
	return Value{
		val:   f / f2,
		vType: Number,
	}, NoError
}

func (v Value) Mod(v2 Value) (Value, int) {
	f, err := v.Float()
	if err != NoError {
		return Value{}, err
	}
	f2, err := v2.Float()
	if err != NoError {
		return Value{}, err
	}
	return Value{
		val:   float64(int(f) % int(f2)),
		vType: Number,
	}, NoError
}
