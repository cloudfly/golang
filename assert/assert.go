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

type KV interface {
	Get(key string) interface{}
}

type symbol struct {
	v        Value
	t        int
	variable bool   // 是否是一个变量
	raw      string // 原生的字符串内容, 即表达式中的符号
}

func newSymbol(s string) symbol {
	item := symbol{
		raw: s,
		v:   nilValue,
		t:   VALUE,
	}
	switch {
	case len(s) >= 2 && (s[0] == '"' || s[0] == '\'' || s[0] == '`'):
		item.v = NewValue("", s[1:len(s)-1])
	case s == "(":
		item.t = LB
	case s == ")":
		item.t = RB
	case s == "!":
		item.t = NOT
	case s == "&&":
		item.t = AND
	case s == "||":
		item.t = OR
	case s == "=":
		item.t = MATCH
	case s == "==":
		item.t = E
	case s == "=~":
		item.t = RE
	case s == "!~":
		item.t = NRE
	case s == "!=":
		item.t = NE
	case s == ">":
		item.t = GT
	case s == "<":
		item.t = LT
	case s == ">=":
		item.t = GTE
	case s == "<=":
		item.t = LTE
	case len(s) >= 1 && unicode.IsNumber(rune(s[0])):
		f, err := strconv.ParseFloat(s, 64)
		if err == nil {
			item.v = NewValue("", f)
		} else {
			item.t = EOF
		}
	case len(s) == 1 && !unicode.IsLetter(rune(s[0])): // +, -, *, /, %
		item.t = int(s[0])
	case s == "true":
		item.v = trueValue
	case s == "false":
		item.v = falseValue
	case s == "nil":
		item.v = nilValue
	default:
		item.variable = true
	}
	return item
}

// Assert 代表一个表达式
type Assert struct {
	*sync.Mutex
	data   []symbol
	pos    int
	answer Value
	kv     KV
	err    error
}

// New 编译代码生成表达式
func New(code string) (*Assert, error) {
	items, err := parse(strings.TrimSpace(code))
	if err != nil {
		return nil, err
	}
	return &Assert{
		Mutex: &sync.Mutex{},
		data:  items,
	}, nil
}

// Lex 为 yacc 使用
func (l *Assert) Lex(lval *yySymType) int {
	if l.pos >= len(l.data) {
		return EOF
	}
	sb := l.data[l.pos]
	l.pos++

	if sb.t == VALUE {
		if sb.variable {
			if l.kv != nil {
				switch instance := l.kv.(type) {
				case pairs:
					lval.value = instance.getValue(sb.raw)
				default:
					lval.value = NewValue(sb.raw, l.kv.Get(sb.raw))
				}
			}
		} else {
			lval.value = sb.v
		}
	}
	return sb.t
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
func (l *Assert) Execute(kv KV) (bool, error) {
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

func parse(data string) ([]symbol, error) {
	items := make([]symbol, 0, 8)
	var (
		// prevState         = 0
		state, start, cut = 0, 0, false
		err               error
	)
	for i := 0; i < len(data); {
		// prevState = state
		state, cut, err = nextState(state, rune(data[i]))
		if err != nil {
			return nil, errors.Wrap(err, "syntax error")
		} else if cut {
			unit := strings.TrimSpace(data[start:i])
			items = append(items, newSymbol(unit))
			/*
				if prevState == 10 { // 10 表示是个变量
				}
			*/
			start = i
		} else {
			i++
		}
	}
	items = append(items, newSymbol(strings.TrimSpace(data[start:])))
	return items, nil
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

type pair struct {
	key   string
	value Value
}
type pairs []pair

// Get 获取变量值
func (p pairs) Get(key string) interface{} {
	for _, item := range p {
		if item.key == key {
			return item.value
		}
	}
	return nil
}

func (p pairs) getValue(key string) Value {
	for _, item := range p {
		if item.key == key {
			return item.value
		}
	}
	return nilValue
}

// NewKV is inner default KV
func NewKV(data map[string]interface{}) KV {
	arr := make(pairs, len(data))
	i := 0
	for k, v := range data {
		arr[i] = pair{k, NewValue(k, v)}
		i++
	}
	return arr
}

// Execute directly run the code with the reader
func Execute(code string, reader KV) (bool, error) {
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
	return exp.Execute(NewKV(data))
}

// MustExecute is same with Execute, but panic if has error
func MustExecute(code string, reader KV) bool {
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
	b, err := exp.Execute(NewKV(data))
	if err != nil {
		panic(err)
	}
	return b
}

// Equal check if the two assert expression is equal, (mainly ignore the white space charactors)
func Equal(code1, code2 string) bool {
	items1, err := parse(strings.TrimSpace(code1))
	if err != nil {
		return false
	}
	items2, err := parse(strings.TrimSpace(code2))
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
