//line yacc.y:2
package assert

import __yyfmt__ "fmt"

//line yacc.y:3
//line yacc.y:7
type yySymType struct {
	yys   int
	value Value
	op    string
}

const VALUE = 57346
const AND = 57347
const OR = 57348
const NOT = 57349
const LB = 57350
const RB = 57351
const E = 57352
const NE = 57353
const RE = 57354
const NRE = 57355
const LT = 57356
const GT = 57357
const LTE = 57358
const GTE = 57359
const EOF = 57360
const MATCH = 57361

var yyToknames = [...]string{
	"$end",
	"error",
	"$unk",
	"VALUE",
	"AND",
	"OR",
	"NOT",
	"LB",
	"RB",
	"E",
	"NE",
	"RE",
	"NRE",
	"LT",
	"GT",
	"LTE",
	"GTE",
	"EOF",
	"MATCH",
	"'+'",
	"'-'",
	"'*'",
	"'/'",
	"'%'",
}
var yyStatenames = [...]string{}

const yyEofCode = 1
const yyErrCode = 2
const yyInitialStackSize = 16

//line yacc.y:129

//line yacctab:1
var yyExca = [...]int{
	-1, 1,
	1, -1,
	-2, 0,
}

const yyPrivate = 57344

const yyLast = 123

var yyAct = [...]int{

	2, 21, 22, 23, 24, 25, 26, 1, 0, 27,
	28, 29, 30, 31, 32, 33, 34, 35, 36, 37,
	38, 39, 40, 41, 42, 8, 9, 0, 0, 43,
	10, 13, 11, 12, 14, 15, 16, 17, 0, 18,
	19, 20, 21, 22, 23, 8, 9, 0, 0, 0,
	10, 13, 11, 12, 14, 15, 16, 17, 7, 18,
	19, 20, 21, 22, 23, 8, 0, 0, 0, 0,
	10, 13, 11, 12, 14, 15, 16, 17, 0, 18,
	19, 20, 21, 22, 23, 10, 13, 11, 12, 14,
	15, 16, 17, 0, 18, 19, 20, 21, 22, 23,
	19, 20, 21, 22, 23, 6, 0, 0, 4, 3,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 5,
}
var yyPact = [...]int{

	101, -1000, 40, 101, 101, 101, -1000, -1000, 101, 101,
	101, 101, 101, 101, 101, 101, 101, 101, 101, 101,
	101, 101, 101, 101, 20, -1000, -21, 75, 60, 80,
	80, 80, 80, 80, 80, 80, 80, 80, -21, -21,
	-1000, -1000, -1000, -1000,
}
var yyPgo = [...]int{

	0, 7, 0,
}
var yyR1 = [...]int{

	0, 1, 2, 2, 2, 2, 2, 2, 2, 2,
	2, 2, 2, 2, 2, 2, 2, 2, 2, 2,
	2, 2,
}
var yyR2 = [...]int{

	0, 2, 3, 2, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	2, 1,
}
var yyChk = [...]int{

	-1000, -1, -2, 8, 7, 21, 4, 18, 5, 6,
	10, 12, 13, 11, 14, 15, 16, 17, 19, 20,
	21, 22, 23, 24, -2, -2, -2, -2, -2, -2,
	-2, -2, -2, -2, -2, -2, -2, -2, -2, -2,
	-2, -2, -2, 9,
}
var yyDef = [...]int{

	0, -2, 0, 0, 0, 0, 21, 1, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 3, 20, 4, 5, 6,
	7, 8, 9, 10, 11, 12, 13, 14, 15, 16,
	17, 18, 19, 2,
}
var yyTok1 = [...]int{

	1, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 24, 3, 3,
	3, 3, 22, 20, 3, 21, 3, 23,
}
var yyTok2 = [...]int{

	2, 3, 4, 5, 6, 7, 8, 9, 10, 11,
	12, 13, 14, 15, 16, 17, 18, 19,
}
var yyTok3 = [...]int{
	0,
}

var yyErrorMessages = [...]struct {
	state int
	token int
	msg   string
}{}

//line yaccpar:1

/*	parser for yacc output	*/

var (
	yyDebug        = 0
	yyErrorVerbose = false
)

type yyLexer interface {
	Lex(lval *yySymType) int
	Error(s string)
}

type yyParser interface {
	Parse(yyLexer) int
	Lookahead() int
}

type yyParserImpl struct {
	lval  yySymType
	stack [yyInitialStackSize]yySymType
	char  int
}

func (p *yyParserImpl) Lookahead() int {
	return p.char
}

func yyNewParser() yyParser {
	return &yyParserImpl{}
}

const yyFlag = -1000

func yyTokname(c int) string {
	if c >= 1 && c-1 < len(yyToknames) {
		if yyToknames[c-1] != "" {
			return yyToknames[c-1]
		}
	}
	return __yyfmt__.Sprintf("tok-%v", c)
}

func yyStatname(s int) string {
	if s >= 0 && s < len(yyStatenames) {
		if yyStatenames[s] != "" {
			return yyStatenames[s]
		}
	}
	return __yyfmt__.Sprintf("state-%v", s)
}

func yyErrorMessage(state, lookAhead int) string {
	const TOKSTART = 4

	if !yyErrorVerbose {
		return "syntax error"
	}

	for _, e := range yyErrorMessages {
		if e.state == state && e.token == lookAhead {
			return "syntax error: " + e.msg
		}
	}

	res := "syntax error: unexpected " + yyTokname(lookAhead)

	// To match Bison, suggest at most four expected tokens.
	expected := make([]int, 0, 4)

	// Look for shiftable tokens.
	base := yyPact[state]
	for tok := TOKSTART; tok-1 < len(yyToknames); tok++ {
		if n := base + tok; n >= 0 && n < yyLast && yyChk[yyAct[n]] == tok {
			if len(expected) == cap(expected) {
				return res
			}
			expected = append(expected, tok)
		}
	}

	if yyDef[state] == -2 {
		i := 0
		for yyExca[i] != -1 || yyExca[i+1] != state {
			i += 2
		}

		// Look for tokens that we accept or reduce.
		for i += 2; yyExca[i] >= 0; i += 2 {
			tok := yyExca[i]
			if tok < TOKSTART || yyExca[i+1] == 0 {
				continue
			}
			if len(expected) == cap(expected) {
				return res
			}
			expected = append(expected, tok)
		}

		// If the default action is to accept or reduce, give up.
		if yyExca[i+1] != 0 {
			return res
		}
	}

	for i, tok := range expected {
		if i == 0 {
			res += ", expecting "
		} else {
			res += " or "
		}
		res += yyTokname(tok)
	}
	return res
}

func yylex1(lex yyLexer, lval *yySymType) (char, token int) {
	token = 0
	char = lex.Lex(lval)
	if char <= 0 {
		token = yyTok1[0]
		goto out
	}
	if char < len(yyTok1) {
		token = yyTok1[char]
		goto out
	}
	if char >= yyPrivate {
		if char < yyPrivate+len(yyTok2) {
			token = yyTok2[char-yyPrivate]
			goto out
		}
	}
	for i := 0; i < len(yyTok3); i += 2 {
		token = yyTok3[i+0]
		if token == char {
			token = yyTok3[i+1]
			goto out
		}
	}

out:
	if token == 0 {
		token = yyTok2[1] /* unknown char */
	}
	if yyDebug >= 3 {
		__yyfmt__.Printf("lex %s(%d)\n", yyTokname(token), uint(char))
	}
	return char, token
}

func yyParse(yylex yyLexer) int {
	return yyNewParser().Parse(yylex)
}

func (yyrcvr *yyParserImpl) Parse(yylex yyLexer) int {
	var yyn int
	var yyVAL yySymType
	var yyDollar []yySymType
	_ = yyDollar // silence set and not used
	yyS := yyrcvr.stack[:]

	Nerrs := 0   /* number of errors */
	Errflag := 0 /* error recovery flag */
	yystate := 0
	yyrcvr.char = -1
	yytoken := -1 // yyrcvr.char translated into internal numbering
	defer func() {
		// Make sure we report no lookahead when not parsing.
		yystate = -1
		yyrcvr.char = -1
		yytoken = -1
	}()
	yyp := -1
	goto yystack

ret0:
	return 0

ret1:
	return 1

yystack:
	/* put a state and value onto the stack */
	if yyDebug >= 4 {
		__yyfmt__.Printf("char %v in %v\n", yyTokname(yytoken), yyStatname(yystate))
	}

	yyp++
	if yyp >= len(yyS) {
		nyys := make([]yySymType, len(yyS)*2)
		copy(nyys, yyS)
		yyS = nyys
	}
	yyS[yyp] = yyVAL
	yyS[yyp].yys = yystate

yynewstate:
	yyn = yyPact[yystate]
	if yyn <= yyFlag {
		goto yydefault /* simple state */
	}
	if yyrcvr.char < 0 {
		yyrcvr.char, yytoken = yylex1(yylex, &yyrcvr.lval)
	}
	yyn += yytoken
	if yyn < 0 || yyn >= yyLast {
		goto yydefault
	}
	yyn = yyAct[yyn]
	if yyChk[yyn] == yytoken { /* valid shift */
		yyrcvr.char = -1
		yytoken = -1
		yyVAL = yyrcvr.lval
		yystate = yyn
		if Errflag > 0 {
			Errflag--
		}
		goto yystack
	}

yydefault:
	/* default state action */
	yyn = yyDef[yystate]
	if yyn == -2 {
		if yyrcvr.char < 0 {
			yyrcvr.char, yytoken = yylex1(yylex, &yyrcvr.lval)
		}

		/* look through exception table */
		xi := 0
		for {
			if yyExca[xi+0] == -1 && yyExca[xi+1] == yystate {
				break
			}
			xi += 2
		}
		for xi += 2; ; xi += 2 {
			yyn = yyExca[xi+0]
			if yyn < 0 || yyn == yytoken {
				break
			}
		}
		yyn = yyExca[xi+1]
		if yyn < 0 {
			goto ret0
		}
	}
	if yyn == 0 {
		/* error ... attempt to resume parsing */
		switch Errflag {
		case 0: /* brand new error */
			yylex.Error(yyErrorMessage(yystate, yytoken))
			Nerrs++
			if yyDebug >= 1 {
				__yyfmt__.Printf("%s", yyStatname(yystate))
				__yyfmt__.Printf(" saw %s\n", yyTokname(yytoken))
			}
			fallthrough

		case 1, 2: /* incompletely recovered error ... try again */
			Errflag = 3

			/* find a state where "error" is a legal shift action */
			for yyp >= 0 {
				yyn = yyPact[yyS[yyp].yys] + yyErrCode
				if yyn >= 0 && yyn < yyLast {
					yystate = yyAct[yyn] /* simulate a shift of "error" */
					if yyChk[yystate] == yyErrCode {
						goto yystack
					}
				}

				/* the current p has no shift on "error", pop stack */
				if yyDebug >= 2 {
					__yyfmt__.Printf("error recovery pops state %d\n", yyS[yyp].yys)
				}
				yyp--
			}
			/* there is no state on the stack with an error shift ... abort */
			goto ret1

		case 3: /* no shift yet; clobber input char */
			if yyDebug >= 2 {
				__yyfmt__.Printf("error recovery discards %s\n", yyTokname(yytoken))
			}
			if yytoken == yyEofCode {
				goto ret1
			}
			yyrcvr.char = -1
			yytoken = -1
			goto yynewstate /* try again in the same state */
		}
	}

	/* reduction by production yyn */
	if yyDebug >= 2 {
		__yyfmt__.Printf("reduce %v in:\n\t%v\n", yyn, yyStatname(yystate))
	}

	yynt := yyn
	yypt := yyp
	_ = yypt // guard against "declared and not used"

	yyp -= yyR2[yyn]
	// yyp is now the index of $0. Perform the default action. Iff the
	// reduced production is ε, $1 is possibly out of range.
	if yyp+1 >= len(yyS) {
		nyys := make([]yySymType, len(yyS)*2)
		copy(nyys, yyS)
		yyS = nyys
	}
	yyVAL = yyS[yyp+1]

	/* consult goto table to find next state */
	yyn = yyR1[yyn]
	yyg := yyPgo[yyn]
	yyj := yyg + yyS[yyp].yys + 1

	if yyj >= yyLast {
		yystate = yyAct[yyg]
	} else {
		yystate = yyAct[yyj]
		if yyChk[yystate] != -yyn {
			yystate = yyAct[yyg]
		}
	}
	// dummy call; replaced with literal code
	switch yynt {

	case 1:
		yyDollar = yyS[yypt-2 : yypt+1]
		//line yacc.y:42
		{
			yylex.(*Assert).answer = yyDollar[1].value
			yyVAL.value = yyDollar[1].value
			return 0
		}
	case 2:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line yacc.y:49
		{
			yyVAL.value = yyDollar[2].value
		}
	case 3:
		yyDollar = yyS[yypt-2 : yypt+1]
		//line yacc.y:53
		{
			yyVAL.value = yyDollar[2].value.Not()
		}
	case 4:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line yacc.y:57
		{
			yyVAL.value = yyDollar[1].value.And(yyDollar[3].value)
		}
	case 5:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line yacc.y:61
		{
			yyVAL.value = yyDollar[1].value.Or(yyDollar[3].value)
		}
	case 6:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line yacc.y:65
		{
			yyVAL.value = yyDollar[1].value.E(yyDollar[3].value)
		}
	case 7:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line yacc.y:69
		{
			yyVAL.value = yyDollar[1].value.RE(yyDollar[3].value)
		}
	case 8:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line yacc.y:73
		{
			yyVAL.value = yyDollar[1].value.NRE(yyDollar[3].value)
		}
	case 9:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line yacc.y:77
		{
			yyVAL.value = yyDollar[1].value.NE(yyDollar[3].value)
		}
	case 10:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line yacc.y:81
		{
			yyVAL.value = yyDollar[1].value.LT(yyDollar[3].value)
		}
	case 11:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line yacc.y:85
		{
			yyVAL.value = yyDollar[1].value.GT(yyDollar[3].value)
		}
	case 12:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line yacc.y:89
		{
			yyVAL.value = yyDollar[1].value.LTE(yyDollar[3].value)
		}
	case 13:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line yacc.y:93
		{
			yyVAL.value = yyDollar[1].value.GTE(yyDollar[3].value)
		}
	case 14:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line yacc.y:97
		{
			yyVAL.value = yyDollar[1].value.MATCH(yyDollar[3].value)
		}
	case 15:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line yacc.y:101
		{
			yyVAL.value = yyDollar[1].value.Add(yyDollar[3].value)
		}
	case 16:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line yacc.y:105
		{
			yyVAL.value = yyDollar[1].value.Sub(yyDollar[3].value)
		}
	case 17:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line yacc.y:109
		{
			yyVAL.value = yyDollar[1].value.Multi(yyDollar[3].value)
		}
	case 18:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line yacc.y:113
		{
			yyVAL.value = yyDollar[1].value.Div(yyDollar[3].value)
		}
	case 19:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line yacc.y:117
		{
			yyVAL.value = yyDollar[1].value.Mod(yyDollar[3].value)
		}
	case 20:
		yyDollar = yyS[yypt-2 : yypt+1]
		//line yacc.y:121
		{
			yyVAL.value = NewValue("", 0).Sub(yyDollar[2].value)
		}
	case 21:
		yyDollar = yyS[yypt-1 : yypt+1]
		//line yacc.y:125
		{
			yyVAL.value = yyDollar[1].value
		}
	}
	goto yystack /* stack new state and value */
}
