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

const (
	ErrNotNumber = 1
	CUT          = 100000
)

type kvReader interface {
	Get(key string) interface{}
}

type Assert struct {
	*sync.Mutex
	data   []string
	pos    int
	answer bool
	kv     kvReader
	err    error
}

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

func (l *Assert) Lex(lval *yySymType) int {
	if l.pos >= len(l.data) {
		return EOF
	}
	s := l.data[l.pos]
	l.pos++
	switch {
	case len(s) >= 2 && (s[0] == '"' || s[0] == '\'' || s[0] == '`'):
		lval.value = NewValue(s[1 : len(s)-1])
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
			lval.value = NewValue(f)
			return VALUE
		}
		return EOF
	case len(s) == 1 && !unicode.IsLetter(rune(s[0])): // +, -, *, /, %
		return int(s[0])
	case s == "true":
		lval.value = NewValue(true)
		return VALUE
	case s == "false":
		lval.value = NewValue(false)
		return VALUE
	case s == "nil":
		lval.value = NewValue(nil)
		return VALUE
	default:
		if l.kv == nil {
			lval.value = NewValue(nil)
		} else {
			lval.value = NewValue(l.kv.Get(s))
		}
		return VALUE
	}
}

func (l *Assert) Error(s string) {
	if s != "" {
		l.err = errors.New(s)
		fmt.Fprintf(os.Stderr, "syntax error: %s\n", s)
	}
}

func (assert *Assert) Execute(kv kvReader) (bool, error) {
	assert.Lock()
	defer assert.Unlock()
	assert.kv = kv
	yyParse(assert)
	assert.pos = 0 // reset the pos
	return assert.answer, assert.err
}

func (assert *Assert) DataAndPos() ([]string, int) {
	ret := make([]string, len(assert.data))
	copy(ret, assert.data)
	return ret, assert.pos
}

func parse(data string) ([]string, error) {
	items := make([]string, 0, 256)
	var (
		state, start, cut = 0, 0, false
		err               error
	)
	for i := 0; i < len(data); {
		state, cut, err = nextState(state, rune(data[i]))
		if err != nil {
			return nil, errors.Wrap(err, "syntax error")
		} else if cut {
			items = append(items, strings.TrimSpace(data[start:i]))
			start = i
		} else {
			i++
		}
	}
	items = append(items, strings.TrimSpace(data[start:]))
	return items, nil
}

const (
	SStart = iota
	SNumber
	SFloat
	Sor
	SAnd
)

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
	case 6: // =, only accept =, to make ==
		if c == '=' || c == '~' {
			return CUT, false, nil
		}
		return 0, false, fmt.Errorf("illegle letter '%c' behind '=', should be '=' or '~'", c)
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
