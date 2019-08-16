package assert

//go:generate goyacc -o yacc.go yacc.y

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"unicode"

	"github.com/pkg/errors"
)

// CUT code
const (
	CUT = 100000
)

type kvReader interface {
	Get(key string) interface{}
}

// Assert 代表一个表达式
type Assert struct {
	*sync.Mutex
	data      []string
	variables []string
	pos       int
	answer    Value
	kv        kvReader
	err       error
}

// New 编译代码生成表达式
func New(code string) (*Assert, error) {
	items, variables, err := parse(strings.TrimSpace(code))
	if err != nil {
		return nil, err
	}
	return &Assert{
		Mutex:     &sync.Mutex{},
		data:      items,
		variables: variables,
	}, nil
}

// Lex 为 yacc 使用
func (l *Assert) Lex(lval *yySymType) int {
	if l.pos >= len(l.data) {
		return EOF
	}
	s := l.data[l.pos]
	l.pos++
	switch {
	case len(s) >= 2 && (s[0] == '"' || s[0] == '\'' || s[0] == '`'):
		lval.value = NewValue("", s[1:len(s)-1])
		return VALUE
	case s == "(":
		return LB
	case s == ")":
		return RB
	case s == "!":
		return NOT
	case s == "&&":
		return AND
	case s == "||":
		return OR
	case s == "=":
		return MATCH
	case s == "==":
		return E
	case s == "=~":
		return RE
	case s == "!~":
		return NRE
	case s == "!=":
		return NE
	case s == ">":
		return GT
	case s == "<":
		return LT
	case s == ">=":
		return GTE
	case s == "<=":
		return LTE
	case len(s) >= 1 && unicode.IsNumber(rune(s[0])):
		f, err := strconv.ParseFloat(s, 64)
		if err == nil {
			lval.value = NewValue("", f)
			return VALUE
		}
		return EOF
	case len(s) == 1 && !unicode.IsLetter(rune(s[0])): // +, -, *, /, %
		return int(s[0])
	case s == "true":
		lval.value = NewValue("", true)
		return VALUE
	case s == "false":
		lval.value = NewValue("", false)
		return VALUE
	case s == "nil":
		lval.value = NewValue("", nil)
		return VALUE
	default:
		if l.kv == nil {
			lval.value = NewValue("", nil)
		} else {
			lval.value = NewValue(s, l.kv.Get(s))
		}
		return VALUE
	}
}

// Error 为 yacc 所用
func (l *Assert) Error(s string) {
	if s != "" {
		l.err = errors.New(s)
		fmt.Fprintf(os.Stderr, "syntax error: %s\n", s)
	}
}

// Execute 使用参数中给定的变量, 执行表达式并返回结果
// 执行过程中出现任何错误都会返回 error, 比如字符串与数字比较等等
func (l *Assert) Execute(kv kvReader) (bool, error) {
	l.Lock()
	defer l.Unlock()
	l.kv = kv
	yyParse(l)
	l.pos = 0 // reset the pos
	if l.err != nil {
		return false, l.err
	}
	if err := l.answer.Error(); err != nil {
		return false, err
	}
	return l.answer.Boolean(), nil
}

// DataAndPos 返回表达式代码中的各个单元
func (l *Assert) DataAndPos() ([]string, int) {
	ret := make([]string, len(l.data))
	copy(ret, l.data)
	return ret, l.pos
}

// Variables 返回表达式中的变量名列表
func (l *Assert) Variables() []string {
	ret := make([]string, len(l.variables))
	copy(ret, l.variables)
	return ret
}

func parse(data string) ([]string, []string, error) {
	items := make([]string, 0, 8)
	variables := make([]string, 0, 4)
	var (
		prevState         = 0
		state, start, cut = 0, 0, false
		err               error
	)
	for i := 0; i < len(data); {
		prevState = state
		state, cut, err = nextState(state, rune(data[i]))
		if err != nil {
			return nil, nil, errors.Wrap(err, "syntax error")
		} else if cut {
			unit := strings.TrimSpace(data[start:i])
			items = append(items, unit)
			if prevState == 10 { // 10 表示是个变量
				variables = append(variables, unit)
			}
			start = i
		} else {
			i++
		}
	}
	items = append(items, strings.TrimSpace(data[start:]))
	return items, variables, nil
}

// return nextState, cutOrNot, errorMessage
// if c should be included into current symbol, return (CUT, false, nil)
// if c should be included into next symbol, return (0, true, nil)
func nextState(state int, c rune) (int, bool, error) {
	switch state {
	case 0:
		if unicode.IsSpace(c) {
			return 0, false, nil
		} else if unicode.IsNumber(c) {
			return 1, false, nil
		}

		switch c {
		case '+', '-', '*', '/', '%', '(', ')':
			return CUT, false, nil
		case '|': // ||
			return 3, false, nil
		case '&': // &&
			return 4, false, nil
		case '!', '>', '<': // !, >, <, !=, >=, <=, !~
			return 5, false, nil
		case '=': // ==
			return 6, false, nil
		case '"': // string in ""
			return 7, false, nil
		case '\'': // string in ''
			return 8, false, nil
		case '`': // string in ``
			return 9, false, nil
		default:
			if unicode.IsLetter(c) || c == '_' || c == '.' {
				return 10, false, nil
			}
			return 0, false, errors.Errorf("illegle charactor '%c'", c)
		}
	case 1: // integer
		if unicode.IsNumber(c) {
			return 1, false, nil
		} else if c == '.' {
			return 2, false, nil
		}
		return 0, true, nil
	case 2: // float
		if unicode.IsNumber(c) {
			return 2, false, nil
		}
		return 0, true, nil
	case 3: // ||
		if c == '|' {
			return CUT, false, nil
		}
		return 0, false, errors.Errorf("illegle letter '%c' behind '|', should be '|'", c)
	case 4: // &&
		if c == '&' {
			return CUT, false, nil
		}
		return 0, false, errors.Errorf("illegle letter '%c' behind '&', should be '&'", c)
	case 5:
		if c == '=' || c == '~' {
			// !=, >=, <=, !~
			return CUT, false, nil
		}
		// !, >, <
		return 0, true, nil
	case 6: // =， ==， =~
		if c == '=' || c == '~' {
			// ==, =~
			return CUT, false, nil
		}
		// =
		return 0, true, nil
	case 7: // string in ""
		if c == '"' {
			return CUT, false, nil
		}
		return 7, false, nil
	case 8: // string in ''
		if c == '\'' {
			return CUT, false, nil
		}
		return 8, false, nil
	case 9: // string in ``
		if c == '`' {
			return CUT, false, nil
		}
		return 9, false, nil
	case 10: // variables
		if unicode.IsLetter(c) || unicode.IsNumber(c) || c == '_' || c == '.' {
			return 10, false, nil
		}
		return 0, true, nil
	case CUT:
		return 0, true, nil
	}
	return 0, false, errors.Errorf("unknown state %d", state)
}

// MapKV 以 map 为基础实现 kvReader
type MapKV map[string]interface{}

// Get 获取变量值
func (kv MapKV) Get(key string) interface{} {
	v, _ := kv[key]
	return v
}

// Execute directly run the code with the reader
func Execute(code string, reader kvReader) (bool, error) {
	exp, err := New(code)
	if err != nil {
		return false, err
	}
	return exp.Execute(reader)
}

// ExecuteMap is a convenient function to execute code with data
func ExecuteMap(code string, data map[string]interface{}) (bool, error) {
	exp, err := New(code)
	if err != nil {
		return false, err
	}
	return exp.Execute(MapKV(data))
}

// MustExecute is same with Execute, but panic if has error
func MustExecute(code string, reader kvReader) bool {
	exp, err := New(code)
	if err != nil {
		panic(err)
	}
	b, err := exp.Execute(reader)
	if err != nil {
		panic(err)
	}
	return b
}

// MustExecuteMap is same with ExecuteMap, but panic if has error
func MustExecuteMap(code string, data map[string]interface{}) bool {
	exp, err := New(code)
	if err != nil {
		panic(err)
	}
	b, err := exp.Execute(MapKV(data))
	if err != nil {
		panic(err)
	}
	return b
}

// Equal check if the two assert expression is equal, (mainly ignore the white space charactors)
func Equal(code1, code2 string) bool {
	items1, _, err := parse(strings.TrimSpace(code1))
	if err != nil {
		return false
	}
	items2, _, err := parse(strings.TrimSpace(code2))
	if err != nil {
		return false
	}
	if len(items1) != len(items2) {
		return false
	}
	for i := 0; i < len(items1); i++ {
		if items1[i] != items2[i] {
			return false
		}
	}
	return true
}
