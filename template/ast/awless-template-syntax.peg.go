package ast

import (
	"fmt"
	"math"
	"sort"
	"strconv"
)

const endSymbol rune = 1114112

/* The rule types inferred from the grammar are below. */
type pegRule uint8

const (
	ruleUnknown pegRule = iota
	ruleScript
	ruleStatement
	ruleAction
	ruleEntity
	ruleVarDeclaration
	ruleDeclaration
	ruleExpr
	ruleParams
	ruleParam
	ruleIdentifier
	ruleValue
	ruleVarValue
	ruleStringValue
	ruleCidrValue
	ruleIpValue
	ruleIntValue
	ruleIntRangeValue
	ruleRefValue
	ruleAliasValue
	ruleHoleValue
	ruleComment
	ruleSpacing
	ruleWhiteSpacing
	ruleMustWhiteSpacing
	ruleEqual
	ruleVar
	ruleSpace
	ruleWhitespace
	ruleEndOfLine
	ruleEndOfFile
	rulePegText
	ruleAction0
	ruleAction1
	ruleAction2
	ruleAction3
	ruleAction4
	ruleAction5
	ruleAction6
	ruleAction7
	ruleAction8
	ruleAction9
	ruleAction10
	ruleAction11
	ruleAction12
	ruleAction13
	ruleAction14
	ruleAction15
	ruleAction16
	ruleAction17
	ruleAction18
	ruleAction19
	ruleAction20
	ruleAction21
)

var rul3s = [...]string{
	"Unknown",
	"Script",
	"Statement",
	"Action",
	"Entity",
	"VarDeclaration",
	"Declaration",
	"Expr",
	"Params",
	"Param",
	"Identifier",
	"Value",
	"VarValue",
	"StringValue",
	"CidrValue",
	"IpValue",
	"IntValue",
	"IntRangeValue",
	"RefValue",
	"AliasValue",
	"HoleValue",
	"Comment",
	"Spacing",
	"WhiteSpacing",
	"MustWhiteSpacing",
	"Equal",
	"Var",
	"Space",
	"Whitespace",
	"EndOfLine",
	"EndOfFile",
	"PegText",
	"Action0",
	"Action1",
	"Action2",
	"Action3",
	"Action4",
	"Action5",
	"Action6",
	"Action7",
	"Action8",
	"Action9",
	"Action10",
	"Action11",
	"Action12",
	"Action13",
	"Action14",
	"Action15",
	"Action16",
	"Action17",
	"Action18",
	"Action19",
	"Action20",
	"Action21",
}

type token32 struct {
	pegRule
	begin, end uint32
}

func (t *token32) String() string {
	return fmt.Sprintf("\x1B[34m%v\x1B[m %v %v", rul3s[t.pegRule], t.begin, t.end)
}

type node32 struct {
	token32
	up, next *node32
}

func (node *node32) print(pretty bool, buffer string) {
	var print func(node *node32, depth int)
	print = func(node *node32, depth int) {
		for node != nil {
			for c := 0; c < depth; c++ {
				fmt.Printf(" ")
			}
			rule := rul3s[node.pegRule]
			quote := strconv.Quote(string(([]rune(buffer)[node.begin:node.end])))
			if !pretty {
				fmt.Printf("%v %v\n", rule, quote)
			} else {
				fmt.Printf("\x1B[34m%v\x1B[m %v\n", rule, quote)
			}
			if node.up != nil {
				print(node.up, depth+1)
			}
			node = node.next
		}
	}
	print(node, 0)
}

func (node *node32) Print(buffer string) {
	node.print(false, buffer)
}

func (node *node32) PrettyPrint(buffer string) {
	node.print(true, buffer)
}

type tokens32 struct {
	tree []token32
}

func (t *tokens32) Trim(length uint32) {
	t.tree = t.tree[:length]
}

func (t *tokens32) Print() {
	for _, token := range t.tree {
		fmt.Println(token.String())
	}
}

func (t *tokens32) AST() *node32 {
	type element struct {
		node *node32
		down *element
	}
	tokens := t.Tokens()
	var stack *element
	for _, token := range tokens {
		if token.begin == token.end {
			continue
		}
		node := &node32{token32: token}
		for stack != nil && stack.node.begin >= token.begin && stack.node.end <= token.end {
			stack.node.next = node.up
			node.up = stack.node
			stack = stack.down
		}
		stack = &element{node: node, down: stack}
	}
	if stack != nil {
		return stack.node
	}
	return nil
}

func (t *tokens32) PrintSyntaxTree(buffer string) {
	t.AST().Print(buffer)
}

func (t *tokens32) PrettyPrintSyntaxTree(buffer string) {
	t.AST().PrettyPrint(buffer)
}

func (t *tokens32) Add(rule pegRule, begin, end, index uint32) {
	if tree := t.tree; int(index) >= len(tree) {
		expanded := make([]token32, 2*len(tree))
		copy(expanded, tree)
		t.tree = expanded
	}
	t.tree[index] = token32{
		pegRule: rule,
		begin:   begin,
		end:     end,
	}
}

func (t *tokens32) Tokens() []token32 {
	return t.tree
}

type Peg struct {
	*AST

	Buffer string
	buffer []rune
	rules  [54]func() bool
	parse  func(rule ...int) error
	reset  func()
	Pretty bool
	tokens32
}

func (p *Peg) Parse(rule ...int) error {
	return p.parse(rule...)
}

func (p *Peg) Reset() {
	p.reset()
}

type textPosition struct {
	line, symbol int
}

type textPositionMap map[int]textPosition

func translatePositions(buffer []rune, positions []int) textPositionMap {
	length, translations, j, line, symbol := len(positions), make(textPositionMap, len(positions)), 0, 1, 0
	sort.Ints(positions)

search:
	for i, c := range buffer {
		if c == '\n' {
			line, symbol = line+1, 0
		} else {
			symbol++
		}
		if i == positions[j] {
			translations[positions[j]] = textPosition{line, symbol}
			for j++; j < length; j++ {
				if i != positions[j] {
					continue search
				}
			}
			break search
		}
	}

	return translations
}

type parseError struct {
	p   *Peg
	max token32
}

func (e *parseError) Error() string {
	tokens, error := []token32{e.max}, "\n"
	positions, p := make([]int, 2*len(tokens)), 0
	for _, token := range tokens {
		positions[p], p = int(token.begin), p+1
		positions[p], p = int(token.end), p+1
	}
	translations := translatePositions(e.p.buffer, positions)
	format := "parse error near %v (line %v symbol %v - line %v symbol %v):\n%v\n"
	if e.p.Pretty {
		format = "parse error near \x1B[34m%v\x1B[m (line %v symbol %v - line %v symbol %v):\n%v\n"
	}
	for _, token := range tokens {
		begin, end := int(token.begin), int(token.end)
		error += fmt.Sprintf(format,
			rul3s[token.pegRule],
			translations[begin].line, translations[begin].symbol,
			translations[end].line, translations[end].symbol,
			strconv.Quote(string(e.p.buffer[begin:end])))
	}

	return error
}

func (p *Peg) PrintSyntaxTree() {
	if p.Pretty {
		p.tokens32.PrettyPrintSyntaxTree(p.Buffer)
	} else {
		p.tokens32.PrintSyntaxTree(p.Buffer)
	}
}

func (p *Peg) Execute() {
	buffer, _buffer, text, begin, end := p.Buffer, p.buffer, "", 0, 0
	for _, token := range p.Tokens() {
		switch token.pegRule {

		case rulePegText:
			begin, end = int(token.begin), int(token.end)
			text = string(_buffer[begin:end])

		case ruleAction0:
			p.AddVarIdentifier(text)
		case ruleAction1:
			p.LineDone()
		case ruleAction2:
			p.AddDeclarationIdentifier(text)
		case ruleAction3:
			p.AddAction(text)
		case ruleAction4:
			p.AddEntity(text)
		case ruleAction5:
			p.LineDone()
		case ruleAction6:
			p.AddParamKey(text)
		case ruleAction7:
			p.AddParamHoleValue(text)
		case ruleAction8:
			p.AddParamAliasValue(text)
		case ruleAction9:
			p.AddParamRefValue(text)
		case ruleAction10:
			p.AddParamCidrValue(text)
		case ruleAction11:
			p.AddParamIpValue(text)
		case ruleAction12:
			p.AddParamValue(text)
		case ruleAction13:
			p.AddParamIntValue(text)
		case ruleAction14:
			p.AddParamValue(text)
		case ruleAction15:
			p.AddVarHoleValue(text)
		case ruleAction16:
			p.AddVarCidrValue(text)
		case ruleAction17:
			p.AddVarIpValue(text)
		case ruleAction18:
			p.AddVarValue(text)
		case ruleAction19:
			p.AddVarIntValue(text)
		case ruleAction20:
			p.AddVarValue(text)
		case ruleAction21:
			p.LineDone()

		}
	}
	_, _, _, _, _ = buffer, _buffer, text, begin, end
}

func (p *Peg) Init() {
	var (
		max                  token32
		position, tokenIndex uint32
		buffer               []rune
	)
	p.reset = func() {
		max = token32{}
		position, tokenIndex = 0, 0

		p.buffer = []rune(p.Buffer)
		if len(p.buffer) == 0 || p.buffer[len(p.buffer)-1] != endSymbol {
			p.buffer = append(p.buffer, endSymbol)
		}
		buffer = p.buffer
	}
	p.reset()

	_rules := p.rules
	tree := tokens32{tree: make([]token32, math.MaxInt16)}
	p.parse = func(rule ...int) error {
		r := 1
		if len(rule) > 0 {
			r = rule[0]
		}
		matches := p.rules[r]()
		p.tokens32 = tree
		if matches {
			p.Trim(tokenIndex)
			return nil
		}
		return &parseError{p, max}
	}

	add := func(rule pegRule, begin uint32) {
		tree.Add(rule, begin, position, tokenIndex)
		tokenIndex++
		if begin != position && position > max.end {
			max = token32{rule, begin, position}
		}
	}

	matchDot := func() bool {
		if buffer[position] != endSymbol {
			position++
			return true
		}
		return false
	}

	/*matchChar := func(c byte) bool {
		if buffer[position] == c {
			position++
			return true
		}
		return false
	}*/

	/*matchRange := func(lower byte, upper byte) bool {
		if c := buffer[position]; c >= lower && c <= upper {
			position++
			return true
		}
		return false
	}*/

	_rules = [...]func() bool{
		nil,
		/* 0 Script <- <(Spacing Statement+ EndOfFile)> */
		func() bool {
			position0, tokenIndex0 := position, tokenIndex
			{
				position1 := position
				if !_rules[ruleSpacing]() {
					goto l0
				}
				{
					position4 := position
					if !_rules[ruleSpacing]() {
						goto l0
					}
					{
						position5, tokenIndex5 := position, tokenIndex
						{
							position7 := position
							{
								position8 := position
								if !_rules[ruleSpacing]() {
									goto l6
								}
								if buffer[position] != rune('v') {
									goto l6
								}
								position++
								if buffer[position] != rune('a') {
									goto l6
								}
								position++
								if buffer[position] != rune('r') {
									goto l6
								}
								position++
								if !_rules[ruleSpacing]() {
									goto l6
								}
								add(ruleVar, position8)
							}
							{
								position9 := position
								if !_rules[ruleIdentifier]() {
									goto l6
								}
								add(rulePegText, position9)
							}
							{
								add(ruleAction0, position)
							}
							if !_rules[ruleEqual]() {
								goto l6
							}
							{
								position11 := position
								{
									position12, tokenIndex12 := position, tokenIndex
									if !_rules[ruleHoleValue]() {
										goto l13
									}
									{
										add(ruleAction15, position)
									}
									goto l12
								l13:
									position, tokenIndex = position12, tokenIndex12
									{
										position16 := position
										if !_rules[ruleCidrValue]() {
											goto l15
										}
										add(rulePegText, position16)
									}
									{
										add(ruleAction16, position)
									}
									goto l12
								l15:
									position, tokenIndex = position12, tokenIndex12
									{
										position19 := position
										if !_rules[ruleIpValue]() {
											goto l18
										}
										add(rulePegText, position19)
									}
									{
										add(ruleAction17, position)
									}
									goto l12
								l18:
									position, tokenIndex = position12, tokenIndex12
									{
										position22 := position
										if !_rules[ruleIntRangeValue]() {
											goto l21
										}
										add(rulePegText, position22)
									}
									{
										add(ruleAction18, position)
									}
									goto l12
								l21:
									position, tokenIndex = position12, tokenIndex12
									{
										position25 := position
										if !_rules[ruleIntValue]() {
											goto l24
										}
										add(rulePegText, position25)
									}
									{
										add(ruleAction19, position)
									}
									goto l12
								l24:
									position, tokenIndex = position12, tokenIndex12
									{
										position27 := position
										if !_rules[ruleStringValue]() {
											goto l6
										}
										add(rulePegText, position27)
									}
									{
										add(ruleAction20, position)
									}
								}
							l12:
								add(ruleVarValue, position11)
							}
							{
								add(ruleAction1, position)
							}
							add(ruleVarDeclaration, position7)
						}
						goto l5
					l6:
						position, tokenIndex = position5, tokenIndex5
						if !_rules[ruleExpr]() {
							goto l30
						}
						goto l5
					l30:
						position, tokenIndex = position5, tokenIndex5
						{
							position32 := position
							{
								position33 := position
								if !_rules[ruleIdentifier]() {
									goto l31
								}
								add(rulePegText, position33)
							}
							{
								add(ruleAction2, position)
							}
							if !_rules[ruleEqual]() {
								goto l31
							}
							if !_rules[ruleExpr]() {
								goto l31
							}
							add(ruleDeclaration, position32)
						}
						goto l5
					l31:
						position, tokenIndex = position5, tokenIndex5
						{
							position35 := position
							{
								position36, tokenIndex36 := position, tokenIndex
								if buffer[position] != rune('#') {
									goto l37
								}
								position++
							l38:
								{
									position39, tokenIndex39 := position, tokenIndex
									{
										position40, tokenIndex40 := position, tokenIndex
										if !_rules[ruleEndOfLine]() {
											goto l40
										}
										goto l39
									l40:
										position, tokenIndex = position40, tokenIndex40
									}
									if !matchDot() {
										goto l39
									}
									goto l38
								l39:
									position, tokenIndex = position39, tokenIndex39
								}
								goto l36
							l37:
								position, tokenIndex = position36, tokenIndex36
								if buffer[position] != rune('/') {
									goto l0
								}
								position++
								if buffer[position] != rune('/') {
									goto l0
								}
								position++
							l41:
								{
									position42, tokenIndex42 := position, tokenIndex
									{
										position43, tokenIndex43 := position, tokenIndex
										if !_rules[ruleEndOfLine]() {
											goto l43
										}
										goto l42
									l43:
										position, tokenIndex = position43, tokenIndex43
									}
									if !matchDot() {
										goto l42
									}
									goto l41
								l42:
									position, tokenIndex = position42, tokenIndex42
								}
								{
									add(ruleAction21, position)
								}
							}
						l36:
							add(ruleComment, position35)
						}
					}
				l5:
					if !_rules[ruleSpacing]() {
						goto l0
					}
				l45:
					{
						position46, tokenIndex46 := position, tokenIndex
						if !_rules[ruleEndOfLine]() {
							goto l46
						}
						goto l45
					l46:
						position, tokenIndex = position46, tokenIndex46
					}
					add(ruleStatement, position4)
				}
			l2:
				{
					position3, tokenIndex3 := position, tokenIndex
					{
						position47 := position
						if !_rules[ruleSpacing]() {
							goto l3
						}
						{
							position48, tokenIndex48 := position, tokenIndex
							{
								position50 := position
								{
									position51 := position
									if !_rules[ruleSpacing]() {
										goto l49
									}
									if buffer[position] != rune('v') {
										goto l49
									}
									position++
									if buffer[position] != rune('a') {
										goto l49
									}
									position++
									if buffer[position] != rune('r') {
										goto l49
									}
									position++
									if !_rules[ruleSpacing]() {
										goto l49
									}
									add(ruleVar, position51)
								}
								{
									position52 := position
									if !_rules[ruleIdentifier]() {
										goto l49
									}
									add(rulePegText, position52)
								}
								{
									add(ruleAction0, position)
								}
								if !_rules[ruleEqual]() {
									goto l49
								}
								{
									position54 := position
									{
										position55, tokenIndex55 := position, tokenIndex
										if !_rules[ruleHoleValue]() {
											goto l56
										}
										{
											add(ruleAction15, position)
										}
										goto l55
									l56:
										position, tokenIndex = position55, tokenIndex55
										{
											position59 := position
											if !_rules[ruleCidrValue]() {
												goto l58
											}
											add(rulePegText, position59)
										}
										{
											add(ruleAction16, position)
										}
										goto l55
									l58:
										position, tokenIndex = position55, tokenIndex55
										{
											position62 := position
											if !_rules[ruleIpValue]() {
												goto l61
											}
											add(rulePegText, position62)
										}
										{
											add(ruleAction17, position)
										}
										goto l55
									l61:
										position, tokenIndex = position55, tokenIndex55
										{
											position65 := position
											if !_rules[ruleIntRangeValue]() {
												goto l64
											}
											add(rulePegText, position65)
										}
										{
											add(ruleAction18, position)
										}
										goto l55
									l64:
										position, tokenIndex = position55, tokenIndex55
										{
											position68 := position
											if !_rules[ruleIntValue]() {
												goto l67
											}
											add(rulePegText, position68)
										}
										{
											add(ruleAction19, position)
										}
										goto l55
									l67:
										position, tokenIndex = position55, tokenIndex55
										{
											position70 := position
											if !_rules[ruleStringValue]() {
												goto l49
											}
											add(rulePegText, position70)
										}
										{
											add(ruleAction20, position)
										}
									}
								l55:
									add(ruleVarValue, position54)
								}
								{
									add(ruleAction1, position)
								}
								add(ruleVarDeclaration, position50)
							}
							goto l48
						l49:
							position, tokenIndex = position48, tokenIndex48
							if !_rules[ruleExpr]() {
								goto l73
							}
							goto l48
						l73:
							position, tokenIndex = position48, tokenIndex48
							{
								position75 := position
								{
									position76 := position
									if !_rules[ruleIdentifier]() {
										goto l74
									}
									add(rulePegText, position76)
								}
								{
									add(ruleAction2, position)
								}
								if !_rules[ruleEqual]() {
									goto l74
								}
								if !_rules[ruleExpr]() {
									goto l74
								}
								add(ruleDeclaration, position75)
							}
							goto l48
						l74:
							position, tokenIndex = position48, tokenIndex48
							{
								position78 := position
								{
									position79, tokenIndex79 := position, tokenIndex
									if buffer[position] != rune('#') {
										goto l80
									}
									position++
								l81:
									{
										position82, tokenIndex82 := position, tokenIndex
										{
											position83, tokenIndex83 := position, tokenIndex
											if !_rules[ruleEndOfLine]() {
												goto l83
											}
											goto l82
										l83:
											position, tokenIndex = position83, tokenIndex83
										}
										if !matchDot() {
											goto l82
										}
										goto l81
									l82:
										position, tokenIndex = position82, tokenIndex82
									}
									goto l79
								l80:
									position, tokenIndex = position79, tokenIndex79
									if buffer[position] != rune('/') {
										goto l3
									}
									position++
									if buffer[position] != rune('/') {
										goto l3
									}
									position++
								l84:
									{
										position85, tokenIndex85 := position, tokenIndex
										{
											position86, tokenIndex86 := position, tokenIndex
											if !_rules[ruleEndOfLine]() {
												goto l86
											}
											goto l85
										l86:
											position, tokenIndex = position86, tokenIndex86
										}
										if !matchDot() {
											goto l85
										}
										goto l84
									l85:
										position, tokenIndex = position85, tokenIndex85
									}
									{
										add(ruleAction21, position)
									}
								}
							l79:
								add(ruleComment, position78)
							}
						}
					l48:
						if !_rules[ruleSpacing]() {
							goto l3
						}
					l88:
						{
							position89, tokenIndex89 := position, tokenIndex
							if !_rules[ruleEndOfLine]() {
								goto l89
							}
							goto l88
						l89:
							position, tokenIndex = position89, tokenIndex89
						}
						add(ruleStatement, position47)
					}
					goto l2
				l3:
					position, tokenIndex = position3, tokenIndex3
				}
				{
					position90 := position
					{
						position91, tokenIndex91 := position, tokenIndex
						if !matchDot() {
							goto l91
						}
						goto l0
					l91:
						position, tokenIndex = position91, tokenIndex91
					}
					add(ruleEndOfFile, position90)
				}
				add(ruleScript, position1)
			}
			return true
		l0:
			position, tokenIndex = position0, tokenIndex0
			return false
		},
		/* 1 Statement <- <(Spacing (VarDeclaration / Expr / Declaration / Comment) Spacing EndOfLine*)> */
		nil,
		/* 2 Action <- <(('c' 'r' 'e' 'a' 't' 'e') / ('d' 'e' 'l' 'e' 't' 'e') / ('s' 't' 'a' 'r' 't') / ((&('d') ('d' 'e' 't' 'a' 'c' 'h')) | (&('c') ('c' 'h' 'e' 'c' 'k')) | (&('a') ('a' 't' 't' 'a' 'c' 'h')) | (&('u') ('u' 'p' 'd' 'a' 't' 'e')) | (&('s') ('s' 't' 'o' 'p'))))> */
		nil,
		/* 3 Entity <- <(('v' 'p' 'c') / ('s' 'u' 'b' 'n' 'e' 't') / ('i' 'n' 's' 't' 'a' 'n' 'c' 'e') / ('r' 'o' 'l' 'e') / ('s' 'e' 'c' 'u' 'r' 'i' 't' 'y' 'g' 'r' 'o' 'u' 'p') / ('r' 'o' 'u' 't' 'e' 't' 'a' 'b' 'l' 'e') / ((&('s') ('s' 't' 'o' 'r' 'a' 'g' 'e' 'o' 'b' 'j' 'e' 'c' 't')) | (&('b') ('b' 'u' 'c' 'k' 'e' 't')) | (&('r') ('r' 'o' 'u' 't' 'e')) | (&('i') ('i' 'n' 't' 'e' 'r' 'n' 'e' 't' 'g' 'a' 't' 'e' 'w' 'a' 'y')) | (&('k') ('k' 'e' 'y' 'p' 'a' 'i' 'r')) | (&('p') ('p' 'o' 'l' 'i' 'c' 'y')) | (&('g') ('g' 'r' 'o' 'u' 'p')) | (&('u') ('u' 's' 'e' 'r')) | (&('t') ('t' 'a' 'g' 's')) | (&('v') ('v' 'o' 'l' 'u' 'm' 'e'))))> */
		nil,
		/* 4 VarDeclaration <- <(Var <Identifier> Action0 Equal VarValue Action1)> */
		nil,
		/* 5 Declaration <- <(<Identifier> Action2 Equal Expr)> */
		nil,
		/* 6 Expr <- <(<Action> Action3 MustWhiteSpacing <Entity> Action4 (MustWhiteSpacing Params)? Action5)> */
		func() bool {
			position97, tokenIndex97 := position, tokenIndex
			{
				position98 := position
				{
					position99 := position
					{
						position100 := position
						{
							position101, tokenIndex101 := position, tokenIndex
							if buffer[position] != rune('c') {
								goto l102
							}
							position++
							if buffer[position] != rune('r') {
								goto l102
							}
							position++
							if buffer[position] != rune('e') {
								goto l102
							}
							position++
							if buffer[position] != rune('a') {
								goto l102
							}
							position++
							if buffer[position] != rune('t') {
								goto l102
							}
							position++
							if buffer[position] != rune('e') {
								goto l102
							}
							position++
							goto l101
						l102:
							position, tokenIndex = position101, tokenIndex101
							if buffer[position] != rune('d') {
								goto l103
							}
							position++
							if buffer[position] != rune('e') {
								goto l103
							}
							position++
							if buffer[position] != rune('l') {
								goto l103
							}
							position++
							if buffer[position] != rune('e') {
								goto l103
							}
							position++
							if buffer[position] != rune('t') {
								goto l103
							}
							position++
							if buffer[position] != rune('e') {
								goto l103
							}
							position++
							goto l101
						l103:
							position, tokenIndex = position101, tokenIndex101
							if buffer[position] != rune('s') {
								goto l104
							}
							position++
							if buffer[position] != rune('t') {
								goto l104
							}
							position++
							if buffer[position] != rune('a') {
								goto l104
							}
							position++
							if buffer[position] != rune('r') {
								goto l104
							}
							position++
							if buffer[position] != rune('t') {
								goto l104
							}
							position++
							goto l101
						l104:
							position, tokenIndex = position101, tokenIndex101
							{
								switch buffer[position] {
								case 'd':
									if buffer[position] != rune('d') {
										goto l97
									}
									position++
									if buffer[position] != rune('e') {
										goto l97
									}
									position++
									if buffer[position] != rune('t') {
										goto l97
									}
									position++
									if buffer[position] != rune('a') {
										goto l97
									}
									position++
									if buffer[position] != rune('c') {
										goto l97
									}
									position++
									if buffer[position] != rune('h') {
										goto l97
									}
									position++
									break
								case 'c':
									if buffer[position] != rune('c') {
										goto l97
									}
									position++
									if buffer[position] != rune('h') {
										goto l97
									}
									position++
									if buffer[position] != rune('e') {
										goto l97
									}
									position++
									if buffer[position] != rune('c') {
										goto l97
									}
									position++
									if buffer[position] != rune('k') {
										goto l97
									}
									position++
									break
								case 'a':
									if buffer[position] != rune('a') {
										goto l97
									}
									position++
									if buffer[position] != rune('t') {
										goto l97
									}
									position++
									if buffer[position] != rune('t') {
										goto l97
									}
									position++
									if buffer[position] != rune('a') {
										goto l97
									}
									position++
									if buffer[position] != rune('c') {
										goto l97
									}
									position++
									if buffer[position] != rune('h') {
										goto l97
									}
									position++
									break
								case 'u':
									if buffer[position] != rune('u') {
										goto l97
									}
									position++
									if buffer[position] != rune('p') {
										goto l97
									}
									position++
									if buffer[position] != rune('d') {
										goto l97
									}
									position++
									if buffer[position] != rune('a') {
										goto l97
									}
									position++
									if buffer[position] != rune('t') {
										goto l97
									}
									position++
									if buffer[position] != rune('e') {
										goto l97
									}
									position++
									break
								default:
									if buffer[position] != rune('s') {
										goto l97
									}
									position++
									if buffer[position] != rune('t') {
										goto l97
									}
									position++
									if buffer[position] != rune('o') {
										goto l97
									}
									position++
									if buffer[position] != rune('p') {
										goto l97
									}
									position++
									break
								}
							}

						}
					l101:
						add(ruleAction, position100)
					}
					add(rulePegText, position99)
				}
				{
					add(ruleAction3, position)
				}
				if !_rules[ruleMustWhiteSpacing]() {
					goto l97
				}
				{
					position107 := position
					{
						position108 := position
						{
							position109, tokenIndex109 := position, tokenIndex
							if buffer[position] != rune('v') {
								goto l110
							}
							position++
							if buffer[position] != rune('p') {
								goto l110
							}
							position++
							if buffer[position] != rune('c') {
								goto l110
							}
							position++
							goto l109
						l110:
							position, tokenIndex = position109, tokenIndex109
							if buffer[position] != rune('s') {
								goto l111
							}
							position++
							if buffer[position] != rune('u') {
								goto l111
							}
							position++
							if buffer[position] != rune('b') {
								goto l111
							}
							position++
							if buffer[position] != rune('n') {
								goto l111
							}
							position++
							if buffer[position] != rune('e') {
								goto l111
							}
							position++
							if buffer[position] != rune('t') {
								goto l111
							}
							position++
							goto l109
						l111:
							position, tokenIndex = position109, tokenIndex109
							if buffer[position] != rune('i') {
								goto l112
							}
							position++
							if buffer[position] != rune('n') {
								goto l112
							}
							position++
							if buffer[position] != rune('s') {
								goto l112
							}
							position++
							if buffer[position] != rune('t') {
								goto l112
							}
							position++
							if buffer[position] != rune('a') {
								goto l112
							}
							position++
							if buffer[position] != rune('n') {
								goto l112
							}
							position++
							if buffer[position] != rune('c') {
								goto l112
							}
							position++
							if buffer[position] != rune('e') {
								goto l112
							}
							position++
							goto l109
						l112:
							position, tokenIndex = position109, tokenIndex109
							if buffer[position] != rune('r') {
								goto l113
							}
							position++
							if buffer[position] != rune('o') {
								goto l113
							}
							position++
							if buffer[position] != rune('l') {
								goto l113
							}
							position++
							if buffer[position] != rune('e') {
								goto l113
							}
							position++
							goto l109
						l113:
							position, tokenIndex = position109, tokenIndex109
							if buffer[position] != rune('s') {
								goto l114
							}
							position++
							if buffer[position] != rune('e') {
								goto l114
							}
							position++
							if buffer[position] != rune('c') {
								goto l114
							}
							position++
							if buffer[position] != rune('u') {
								goto l114
							}
							position++
							if buffer[position] != rune('r') {
								goto l114
							}
							position++
							if buffer[position] != rune('i') {
								goto l114
							}
							position++
							if buffer[position] != rune('t') {
								goto l114
							}
							position++
							if buffer[position] != rune('y') {
								goto l114
							}
							position++
							if buffer[position] != rune('g') {
								goto l114
							}
							position++
							if buffer[position] != rune('r') {
								goto l114
							}
							position++
							if buffer[position] != rune('o') {
								goto l114
							}
							position++
							if buffer[position] != rune('u') {
								goto l114
							}
							position++
							if buffer[position] != rune('p') {
								goto l114
							}
							position++
							goto l109
						l114:
							position, tokenIndex = position109, tokenIndex109
							if buffer[position] != rune('r') {
								goto l115
							}
							position++
							if buffer[position] != rune('o') {
								goto l115
							}
							position++
							if buffer[position] != rune('u') {
								goto l115
							}
							position++
							if buffer[position] != rune('t') {
								goto l115
							}
							position++
							if buffer[position] != rune('e') {
								goto l115
							}
							position++
							if buffer[position] != rune('t') {
								goto l115
							}
							position++
							if buffer[position] != rune('a') {
								goto l115
							}
							position++
							if buffer[position] != rune('b') {
								goto l115
							}
							position++
							if buffer[position] != rune('l') {
								goto l115
							}
							position++
							if buffer[position] != rune('e') {
								goto l115
							}
							position++
							goto l109
						l115:
							position, tokenIndex = position109, tokenIndex109
							{
								switch buffer[position] {
								case 's':
									if buffer[position] != rune('s') {
										goto l97
									}
									position++
									if buffer[position] != rune('t') {
										goto l97
									}
									position++
									if buffer[position] != rune('o') {
										goto l97
									}
									position++
									if buffer[position] != rune('r') {
										goto l97
									}
									position++
									if buffer[position] != rune('a') {
										goto l97
									}
									position++
									if buffer[position] != rune('g') {
										goto l97
									}
									position++
									if buffer[position] != rune('e') {
										goto l97
									}
									position++
									if buffer[position] != rune('o') {
										goto l97
									}
									position++
									if buffer[position] != rune('b') {
										goto l97
									}
									position++
									if buffer[position] != rune('j') {
										goto l97
									}
									position++
									if buffer[position] != rune('e') {
										goto l97
									}
									position++
									if buffer[position] != rune('c') {
										goto l97
									}
									position++
									if buffer[position] != rune('t') {
										goto l97
									}
									position++
									break
								case 'b':
									if buffer[position] != rune('b') {
										goto l97
									}
									position++
									if buffer[position] != rune('u') {
										goto l97
									}
									position++
									if buffer[position] != rune('c') {
										goto l97
									}
									position++
									if buffer[position] != rune('k') {
										goto l97
									}
									position++
									if buffer[position] != rune('e') {
										goto l97
									}
									position++
									if buffer[position] != rune('t') {
										goto l97
									}
									position++
									break
								case 'r':
									if buffer[position] != rune('r') {
										goto l97
									}
									position++
									if buffer[position] != rune('o') {
										goto l97
									}
									position++
									if buffer[position] != rune('u') {
										goto l97
									}
									position++
									if buffer[position] != rune('t') {
										goto l97
									}
									position++
									if buffer[position] != rune('e') {
										goto l97
									}
									position++
									break
								case 'i':
									if buffer[position] != rune('i') {
										goto l97
									}
									position++
									if buffer[position] != rune('n') {
										goto l97
									}
									position++
									if buffer[position] != rune('t') {
										goto l97
									}
									position++
									if buffer[position] != rune('e') {
										goto l97
									}
									position++
									if buffer[position] != rune('r') {
										goto l97
									}
									position++
									if buffer[position] != rune('n') {
										goto l97
									}
									position++
									if buffer[position] != rune('e') {
										goto l97
									}
									position++
									if buffer[position] != rune('t') {
										goto l97
									}
									position++
									if buffer[position] != rune('g') {
										goto l97
									}
									position++
									if buffer[position] != rune('a') {
										goto l97
									}
									position++
									if buffer[position] != rune('t') {
										goto l97
									}
									position++
									if buffer[position] != rune('e') {
										goto l97
									}
									position++
									if buffer[position] != rune('w') {
										goto l97
									}
									position++
									if buffer[position] != rune('a') {
										goto l97
									}
									position++
									if buffer[position] != rune('y') {
										goto l97
									}
									position++
									break
								case 'k':
									if buffer[position] != rune('k') {
										goto l97
									}
									position++
									if buffer[position] != rune('e') {
										goto l97
									}
									position++
									if buffer[position] != rune('y') {
										goto l97
									}
									position++
									if buffer[position] != rune('p') {
										goto l97
									}
									position++
									if buffer[position] != rune('a') {
										goto l97
									}
									position++
									if buffer[position] != rune('i') {
										goto l97
									}
									position++
									if buffer[position] != rune('r') {
										goto l97
									}
									position++
									break
								case 'p':
									if buffer[position] != rune('p') {
										goto l97
									}
									position++
									if buffer[position] != rune('o') {
										goto l97
									}
									position++
									if buffer[position] != rune('l') {
										goto l97
									}
									position++
									if buffer[position] != rune('i') {
										goto l97
									}
									position++
									if buffer[position] != rune('c') {
										goto l97
									}
									position++
									if buffer[position] != rune('y') {
										goto l97
									}
									position++
									break
								case 'g':
									if buffer[position] != rune('g') {
										goto l97
									}
									position++
									if buffer[position] != rune('r') {
										goto l97
									}
									position++
									if buffer[position] != rune('o') {
										goto l97
									}
									position++
									if buffer[position] != rune('u') {
										goto l97
									}
									position++
									if buffer[position] != rune('p') {
										goto l97
									}
									position++
									break
								case 'u':
									if buffer[position] != rune('u') {
										goto l97
									}
									position++
									if buffer[position] != rune('s') {
										goto l97
									}
									position++
									if buffer[position] != rune('e') {
										goto l97
									}
									position++
									if buffer[position] != rune('r') {
										goto l97
									}
									position++
									break
								case 't':
									if buffer[position] != rune('t') {
										goto l97
									}
									position++
									if buffer[position] != rune('a') {
										goto l97
									}
									position++
									if buffer[position] != rune('g') {
										goto l97
									}
									position++
									if buffer[position] != rune('s') {
										goto l97
									}
									position++
									break
								default:
									if buffer[position] != rune('v') {
										goto l97
									}
									position++
									if buffer[position] != rune('o') {
										goto l97
									}
									position++
									if buffer[position] != rune('l') {
										goto l97
									}
									position++
									if buffer[position] != rune('u') {
										goto l97
									}
									position++
									if buffer[position] != rune('m') {
										goto l97
									}
									position++
									if buffer[position] != rune('e') {
										goto l97
									}
									position++
									break
								}
							}

						}
					l109:
						add(ruleEntity, position108)
					}
					add(rulePegText, position107)
				}
				{
					add(ruleAction4, position)
				}
				{
					position118, tokenIndex118 := position, tokenIndex
					if !_rules[ruleMustWhiteSpacing]() {
						goto l118
					}
					{
						position120 := position
						{
							position123 := position
							{
								position124 := position
								if !_rules[ruleIdentifier]() {
									goto l118
								}
								add(rulePegText, position124)
							}
							{
								add(ruleAction6, position)
							}
							if !_rules[ruleEqual]() {
								goto l118
							}
							{
								position126 := position
								{
									position127, tokenIndex127 := position, tokenIndex
									{
										position129 := position
										if !_rules[ruleCidrValue]() {
											goto l128
										}
										add(rulePegText, position129)
									}
									{
										add(ruleAction10, position)
									}
									goto l127
								l128:
									position, tokenIndex = position127, tokenIndex127
									{
										position132 := position
										if !_rules[ruleIpValue]() {
											goto l131
										}
										add(rulePegText, position132)
									}
									{
										add(ruleAction11, position)
									}
									goto l127
								l131:
									position, tokenIndex = position127, tokenIndex127
									{
										position135 := position
										if !_rules[ruleIntRangeValue]() {
											goto l134
										}
										add(rulePegText, position135)
									}
									{
										add(ruleAction12, position)
									}
									goto l127
								l134:
									position, tokenIndex = position127, tokenIndex127
									{
										position138 := position
										if !_rules[ruleIntValue]() {
											goto l137
										}
										add(rulePegText, position138)
									}
									{
										add(ruleAction13, position)
									}
									goto l127
								l137:
									position, tokenIndex = position127, tokenIndex127
									{
										switch buffer[position] {
										case '$':
											{
												position141 := position
												if buffer[position] != rune('$') {
													goto l118
												}
												position++
												{
													position142 := position
													if !_rules[ruleIdentifier]() {
														goto l118
													}
													add(rulePegText, position142)
												}
												add(ruleRefValue, position141)
											}
											{
												add(ruleAction9, position)
											}
											break
										case '@':
											{
												position144 := position
												if buffer[position] != rune('@') {
													goto l118
												}
												position++
												{
													position145 := position
													if !_rules[ruleIdentifier]() {
														goto l118
													}
													add(rulePegText, position145)
												}
												add(ruleAliasValue, position144)
											}
											{
												add(ruleAction8, position)
											}
											break
										case '{':
											if !_rules[ruleHoleValue]() {
												goto l118
											}
											{
												add(ruleAction7, position)
											}
											break
										default:
											{
												position148 := position
												if !_rules[ruleStringValue]() {
													goto l118
												}
												add(rulePegText, position148)
											}
											{
												add(ruleAction14, position)
											}
											break
										}
									}

								}
							l127:
								add(ruleValue, position126)
							}
							if !_rules[ruleWhiteSpacing]() {
								goto l118
							}
							add(ruleParam, position123)
						}
					l121:
						{
							position122, tokenIndex122 := position, tokenIndex
							{
								position150 := position
								{
									position151 := position
									if !_rules[ruleIdentifier]() {
										goto l122
									}
									add(rulePegText, position151)
								}
								{
									add(ruleAction6, position)
								}
								if !_rules[ruleEqual]() {
									goto l122
								}
								{
									position153 := position
									{
										position154, tokenIndex154 := position, tokenIndex
										{
											position156 := position
											if !_rules[ruleCidrValue]() {
												goto l155
											}
											add(rulePegText, position156)
										}
										{
											add(ruleAction10, position)
										}
										goto l154
									l155:
										position, tokenIndex = position154, tokenIndex154
										{
											position159 := position
											if !_rules[ruleIpValue]() {
												goto l158
											}
											add(rulePegText, position159)
										}
										{
											add(ruleAction11, position)
										}
										goto l154
									l158:
										position, tokenIndex = position154, tokenIndex154
										{
											position162 := position
											if !_rules[ruleIntRangeValue]() {
												goto l161
											}
											add(rulePegText, position162)
										}
										{
											add(ruleAction12, position)
										}
										goto l154
									l161:
										position, tokenIndex = position154, tokenIndex154
										{
											position165 := position
											if !_rules[ruleIntValue]() {
												goto l164
											}
											add(rulePegText, position165)
										}
										{
											add(ruleAction13, position)
										}
										goto l154
									l164:
										position, tokenIndex = position154, tokenIndex154
										{
											switch buffer[position] {
											case '$':
												{
													position168 := position
													if buffer[position] != rune('$') {
														goto l122
													}
													position++
													{
														position169 := position
														if !_rules[ruleIdentifier]() {
															goto l122
														}
														add(rulePegText, position169)
													}
													add(ruleRefValue, position168)
												}
												{
													add(ruleAction9, position)
												}
												break
											case '@':
												{
													position171 := position
													if buffer[position] != rune('@') {
														goto l122
													}
													position++
													{
														position172 := position
														if !_rules[ruleIdentifier]() {
															goto l122
														}
														add(rulePegText, position172)
													}
													add(ruleAliasValue, position171)
												}
												{
													add(ruleAction8, position)
												}
												break
											case '{':
												if !_rules[ruleHoleValue]() {
													goto l122
												}
												{
													add(ruleAction7, position)
												}
												break
											default:
												{
													position175 := position
													if !_rules[ruleStringValue]() {
														goto l122
													}
													add(rulePegText, position175)
												}
												{
													add(ruleAction14, position)
												}
												break
											}
										}

									}
								l154:
									add(ruleValue, position153)
								}
								if !_rules[ruleWhiteSpacing]() {
									goto l122
								}
								add(ruleParam, position150)
							}
							goto l121
						l122:
							position, tokenIndex = position122, tokenIndex122
						}
						add(ruleParams, position120)
					}
					goto l119
				l118:
					position, tokenIndex = position118, tokenIndex118
				}
			l119:
				{
					add(ruleAction5, position)
				}
				add(ruleExpr, position98)
			}
			return true
		l97:
			position, tokenIndex = position97, tokenIndex97
			return false
		},
		/* 7 Params <- <Param+> */
		nil,
		/* 8 Param <- <(<Identifier> Action6 Equal Value WhiteSpacing)> */
		nil,
		/* 9 Identifier <- <((&('.') '.') | (&('_') '_') | (&('-') '-') | (&('A' | 'B' | 'C' | 'D' | 'E' | 'F' | 'G' | 'H' | 'I' | 'J' | 'K' | 'L' | 'M' | 'N' | 'O' | 'P' | 'Q' | 'R' | 'S' | 'T' | 'U' | 'V' | 'W' | 'X' | 'Y' | 'Z') [A-Z]) | (&('a' | 'b' | 'c' | 'd' | 'e' | 'f' | 'g' | 'h' | 'i' | 'j' | 'k' | 'l' | 'm' | 'n' | 'o' | 'p' | 'q' | 'r' | 's' | 't' | 'u' | 'v' | 'w' | 'x' | 'y' | 'z') [a-z]))+> */
		func() bool {
			position180, tokenIndex180 := position, tokenIndex
			{
				position181 := position
				{
					switch buffer[position] {
					case '.':
						if buffer[position] != rune('.') {
							goto l180
						}
						position++
						break
					case '_':
						if buffer[position] != rune('_') {
							goto l180
						}
						position++
						break
					case '-':
						if buffer[position] != rune('-') {
							goto l180
						}
						position++
						break
					case 'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z':
						if c := buffer[position]; c < rune('A') || c > rune('Z') {
							goto l180
						}
						position++
						break
					default:
						if c := buffer[position]; c < rune('a') || c > rune('z') {
							goto l180
						}
						position++
						break
					}
				}

			l182:
				{
					position183, tokenIndex183 := position, tokenIndex
					{
						switch buffer[position] {
						case '.':
							if buffer[position] != rune('.') {
								goto l183
							}
							position++
							break
						case '_':
							if buffer[position] != rune('_') {
								goto l183
							}
							position++
							break
						case '-':
							if buffer[position] != rune('-') {
								goto l183
							}
							position++
							break
						case 'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z':
							if c := buffer[position]; c < rune('A') || c > rune('Z') {
								goto l183
							}
							position++
							break
						default:
							if c := buffer[position]; c < rune('a') || c > rune('z') {
								goto l183
							}
							position++
							break
						}
					}

					goto l182
				l183:
					position, tokenIndex = position183, tokenIndex183
				}
				add(ruleIdentifier, position181)
			}
			return true
		l180:
			position, tokenIndex = position180, tokenIndex180
			return false
		},
		/* 10 Value <- <((<CidrValue> Action10) / (<IpValue> Action11) / (<IntRangeValue> Action12) / (<IntValue> Action13) / ((&('$') (RefValue Action9)) | (&('@') (AliasValue Action8)) | (&('{') (HoleValue Action7)) | (&('-' | '.' | '/' | '0' | '1' | '2' | '3' | '4' | '5' | '6' | '7' | '8' | '9' | ':' | 'A' | 'B' | 'C' | 'D' | 'E' | 'F' | 'G' | 'H' | 'I' | 'J' | 'K' | 'L' | 'M' | 'N' | 'O' | 'P' | 'Q' | 'R' | 'S' | 'T' | 'U' | 'V' | 'W' | 'X' | 'Y' | 'Z' | '_' | 'a' | 'b' | 'c' | 'd' | 'e' | 'f' | 'g' | 'h' | 'i' | 'j' | 'k' | 'l' | 'm' | 'n' | 'o' | 'p' | 'q' | 'r' | 's' | 't' | 'u' | 'v' | 'w' | 'x' | 'y' | 'z') (<StringValue> Action14))))> */
		nil,
		/* 11 VarValue <- <((HoleValue Action15) / (<CidrValue> Action16) / (<IpValue> Action17) / (<IntRangeValue> Action18) / (<IntValue> Action19) / (<StringValue> Action20))> */
		nil,
		/* 12 StringValue <- <((&('/') '/') | (&(':') ':') | (&('_') '_') | (&('.') '.') | (&('-') '-') | (&('0' | '1' | '2' | '3' | '4' | '5' | '6' | '7' | '8' | '9') [0-9]) | (&('A' | 'B' | 'C' | 'D' | 'E' | 'F' | 'G' | 'H' | 'I' | 'J' | 'K' | 'L' | 'M' | 'N' | 'O' | 'P' | 'Q' | 'R' | 'S' | 'T' | 'U' | 'V' | 'W' | 'X' | 'Y' | 'Z') [A-Z]) | (&('a' | 'b' | 'c' | 'd' | 'e' | 'f' | 'g' | 'h' | 'i' | 'j' | 'k' | 'l' | 'm' | 'n' | 'o' | 'p' | 'q' | 'r' | 's' | 't' | 'u' | 'v' | 'w' | 'x' | 'y' | 'z') [a-z]))+> */
		func() bool {
			position188, tokenIndex188 := position, tokenIndex
			{
				position189 := position
				{
					switch buffer[position] {
					case '/':
						if buffer[position] != rune('/') {
							goto l188
						}
						position++
						break
					case ':':
						if buffer[position] != rune(':') {
							goto l188
						}
						position++
						break
					case '_':
						if buffer[position] != rune('_') {
							goto l188
						}
						position++
						break
					case '.':
						if buffer[position] != rune('.') {
							goto l188
						}
						position++
						break
					case '-':
						if buffer[position] != rune('-') {
							goto l188
						}
						position++
						break
					case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
						if c := buffer[position]; c < rune('0') || c > rune('9') {
							goto l188
						}
						position++
						break
					case 'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z':
						if c := buffer[position]; c < rune('A') || c > rune('Z') {
							goto l188
						}
						position++
						break
					default:
						if c := buffer[position]; c < rune('a') || c > rune('z') {
							goto l188
						}
						position++
						break
					}
				}

			l190:
				{
					position191, tokenIndex191 := position, tokenIndex
					{
						switch buffer[position] {
						case '/':
							if buffer[position] != rune('/') {
								goto l191
							}
							position++
							break
						case ':':
							if buffer[position] != rune(':') {
								goto l191
							}
							position++
							break
						case '_':
							if buffer[position] != rune('_') {
								goto l191
							}
							position++
							break
						case '.':
							if buffer[position] != rune('.') {
								goto l191
							}
							position++
							break
						case '-':
							if buffer[position] != rune('-') {
								goto l191
							}
							position++
							break
						case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
							if c := buffer[position]; c < rune('0') || c > rune('9') {
								goto l191
							}
							position++
							break
						case 'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z':
							if c := buffer[position]; c < rune('A') || c > rune('Z') {
								goto l191
							}
							position++
							break
						default:
							if c := buffer[position]; c < rune('a') || c > rune('z') {
								goto l191
							}
							position++
							break
						}
					}

					goto l190
				l191:
					position, tokenIndex = position191, tokenIndex191
				}
				add(ruleStringValue, position189)
			}
			return true
		l188:
			position, tokenIndex = position188, tokenIndex188
			return false
		},
		/* 13 CidrValue <- <([0-9]+ . [0-9]+ . [0-9]+ . [0-9]+ '/' [0-9]+)> */
		func() bool {
			position194, tokenIndex194 := position, tokenIndex
			{
				position195 := position
				if c := buffer[position]; c < rune('0') || c > rune('9') {
					goto l194
				}
				position++
			l196:
				{
					position197, tokenIndex197 := position, tokenIndex
					if c := buffer[position]; c < rune('0') || c > rune('9') {
						goto l197
					}
					position++
					goto l196
				l197:
					position, tokenIndex = position197, tokenIndex197
				}
				if !matchDot() {
					goto l194
				}
				if c := buffer[position]; c < rune('0') || c > rune('9') {
					goto l194
				}
				position++
			l198:
				{
					position199, tokenIndex199 := position, tokenIndex
					if c := buffer[position]; c < rune('0') || c > rune('9') {
						goto l199
					}
					position++
					goto l198
				l199:
					position, tokenIndex = position199, tokenIndex199
				}
				if !matchDot() {
					goto l194
				}
				if c := buffer[position]; c < rune('0') || c > rune('9') {
					goto l194
				}
				position++
			l200:
				{
					position201, tokenIndex201 := position, tokenIndex
					if c := buffer[position]; c < rune('0') || c > rune('9') {
						goto l201
					}
					position++
					goto l200
				l201:
					position, tokenIndex = position201, tokenIndex201
				}
				if !matchDot() {
					goto l194
				}
				if c := buffer[position]; c < rune('0') || c > rune('9') {
					goto l194
				}
				position++
			l202:
				{
					position203, tokenIndex203 := position, tokenIndex
					if c := buffer[position]; c < rune('0') || c > rune('9') {
						goto l203
					}
					position++
					goto l202
				l203:
					position, tokenIndex = position203, tokenIndex203
				}
				if buffer[position] != rune('/') {
					goto l194
				}
				position++
				if c := buffer[position]; c < rune('0') || c > rune('9') {
					goto l194
				}
				position++
			l204:
				{
					position205, tokenIndex205 := position, tokenIndex
					if c := buffer[position]; c < rune('0') || c > rune('9') {
						goto l205
					}
					position++
					goto l204
				l205:
					position, tokenIndex = position205, tokenIndex205
				}
				add(ruleCidrValue, position195)
			}
			return true
		l194:
			position, tokenIndex = position194, tokenIndex194
			return false
		},
		/* 14 IpValue <- <([0-9]+ . [0-9]+ . [0-9]+ . [0-9]+)> */
		func() bool {
			position206, tokenIndex206 := position, tokenIndex
			{
				position207 := position
				if c := buffer[position]; c < rune('0') || c > rune('9') {
					goto l206
				}
				position++
			l208:
				{
					position209, tokenIndex209 := position, tokenIndex
					if c := buffer[position]; c < rune('0') || c > rune('9') {
						goto l209
					}
					position++
					goto l208
				l209:
					position, tokenIndex = position209, tokenIndex209
				}
				if !matchDot() {
					goto l206
				}
				if c := buffer[position]; c < rune('0') || c > rune('9') {
					goto l206
				}
				position++
			l210:
				{
					position211, tokenIndex211 := position, tokenIndex
					if c := buffer[position]; c < rune('0') || c > rune('9') {
						goto l211
					}
					position++
					goto l210
				l211:
					position, tokenIndex = position211, tokenIndex211
				}
				if !matchDot() {
					goto l206
				}
				if c := buffer[position]; c < rune('0') || c > rune('9') {
					goto l206
				}
				position++
			l212:
				{
					position213, tokenIndex213 := position, tokenIndex
					if c := buffer[position]; c < rune('0') || c > rune('9') {
						goto l213
					}
					position++
					goto l212
				l213:
					position, tokenIndex = position213, tokenIndex213
				}
				if !matchDot() {
					goto l206
				}
				if c := buffer[position]; c < rune('0') || c > rune('9') {
					goto l206
				}
				position++
			l214:
				{
					position215, tokenIndex215 := position, tokenIndex
					if c := buffer[position]; c < rune('0') || c > rune('9') {
						goto l215
					}
					position++
					goto l214
				l215:
					position, tokenIndex = position215, tokenIndex215
				}
				add(ruleIpValue, position207)
			}
			return true
		l206:
			position, tokenIndex = position206, tokenIndex206
			return false
		},
		/* 15 IntValue <- <[0-9]+> */
		func() bool {
			position216, tokenIndex216 := position, tokenIndex
			{
				position217 := position
				if c := buffer[position]; c < rune('0') || c > rune('9') {
					goto l216
				}
				position++
			l218:
				{
					position219, tokenIndex219 := position, tokenIndex
					if c := buffer[position]; c < rune('0') || c > rune('9') {
						goto l219
					}
					position++
					goto l218
				l219:
					position, tokenIndex = position219, tokenIndex219
				}
				add(ruleIntValue, position217)
			}
			return true
		l216:
			position, tokenIndex = position216, tokenIndex216
			return false
		},
		/* 16 IntRangeValue <- <([0-9]+ '-' [0-9]+)> */
		func() bool {
			position220, tokenIndex220 := position, tokenIndex
			{
				position221 := position
				if c := buffer[position]; c < rune('0') || c > rune('9') {
					goto l220
				}
				position++
			l222:
				{
					position223, tokenIndex223 := position, tokenIndex
					if c := buffer[position]; c < rune('0') || c > rune('9') {
						goto l223
					}
					position++
					goto l222
				l223:
					position, tokenIndex = position223, tokenIndex223
				}
				if buffer[position] != rune('-') {
					goto l220
				}
				position++
				if c := buffer[position]; c < rune('0') || c > rune('9') {
					goto l220
				}
				position++
			l224:
				{
					position225, tokenIndex225 := position, tokenIndex
					if c := buffer[position]; c < rune('0') || c > rune('9') {
						goto l225
					}
					position++
					goto l224
				l225:
					position, tokenIndex = position225, tokenIndex225
				}
				add(ruleIntRangeValue, position221)
			}
			return true
		l220:
			position, tokenIndex = position220, tokenIndex220
			return false
		},
		/* 17 RefValue <- <('$' <Identifier>)> */
		nil,
		/* 18 AliasValue <- <('@' <Identifier>)> */
		nil,
		/* 19 HoleValue <- <('{' WhiteSpacing <Identifier> WhiteSpacing '}')> */
		func() bool {
			position228, tokenIndex228 := position, tokenIndex
			{
				position229 := position
				if buffer[position] != rune('{') {
					goto l228
				}
				position++
				if !_rules[ruleWhiteSpacing]() {
					goto l228
				}
				{
					position230 := position
					if !_rules[ruleIdentifier]() {
						goto l228
					}
					add(rulePegText, position230)
				}
				if !_rules[ruleWhiteSpacing]() {
					goto l228
				}
				if buffer[position] != rune('}') {
					goto l228
				}
				position++
				add(ruleHoleValue, position229)
			}
			return true
		l228:
			position, tokenIndex = position228, tokenIndex228
			return false
		},
		/* 20 Comment <- <(('#' (!EndOfLine .)*) / ('/' '/' (!EndOfLine .)* Action21))> */
		nil,
		/* 21 Spacing <- <Space*> */
		func() bool {
			{
				position233 := position
			l234:
				{
					position235, tokenIndex235 := position, tokenIndex
					{
						position236 := position
						{
							position237, tokenIndex237 := position, tokenIndex
							if !_rules[ruleWhitespace]() {
								goto l238
							}
							goto l237
						l238:
							position, tokenIndex = position237, tokenIndex237
							if !_rules[ruleEndOfLine]() {
								goto l235
							}
						}
					l237:
						add(ruleSpace, position236)
					}
					goto l234
				l235:
					position, tokenIndex = position235, tokenIndex235
				}
				add(ruleSpacing, position233)
			}
			return true
		},
		/* 22 WhiteSpacing <- <Whitespace*> */
		func() bool {
			{
				position240 := position
			l241:
				{
					position242, tokenIndex242 := position, tokenIndex
					if !_rules[ruleWhitespace]() {
						goto l242
					}
					goto l241
				l242:
					position, tokenIndex = position242, tokenIndex242
				}
				add(ruleWhiteSpacing, position240)
			}
			return true
		},
		/* 23 MustWhiteSpacing <- <Whitespace+> */
		func() bool {
			position243, tokenIndex243 := position, tokenIndex
			{
				position244 := position
				if !_rules[ruleWhitespace]() {
					goto l243
				}
			l245:
				{
					position246, tokenIndex246 := position, tokenIndex
					if !_rules[ruleWhitespace]() {
						goto l246
					}
					goto l245
				l246:
					position, tokenIndex = position246, tokenIndex246
				}
				add(ruleMustWhiteSpacing, position244)
			}
			return true
		l243:
			position, tokenIndex = position243, tokenIndex243
			return false
		},
		/* 24 Equal <- <(Spacing '=' Spacing)> */
		func() bool {
			position247, tokenIndex247 := position, tokenIndex
			{
				position248 := position
				if !_rules[ruleSpacing]() {
					goto l247
				}
				if buffer[position] != rune('=') {
					goto l247
				}
				position++
				if !_rules[ruleSpacing]() {
					goto l247
				}
				add(ruleEqual, position248)
			}
			return true
		l247:
			position, tokenIndex = position247, tokenIndex247
			return false
		},
		/* 25 Var <- <(Spacing ('v' 'a' 'r') Spacing)> */
		nil,
		/* 26 Space <- <(Whitespace / EndOfLine)> */
		nil,
		/* 27 Whitespace <- <(' ' / '\t')> */
		func() bool {
			position251, tokenIndex251 := position, tokenIndex
			{
				position252 := position
				{
					position253, tokenIndex253 := position, tokenIndex
					if buffer[position] != rune(' ') {
						goto l254
					}
					position++
					goto l253
				l254:
					position, tokenIndex = position253, tokenIndex253
					if buffer[position] != rune('\t') {
						goto l251
					}
					position++
				}
			l253:
				add(ruleWhitespace, position252)
			}
			return true
		l251:
			position, tokenIndex = position251, tokenIndex251
			return false
		},
		/* 28 EndOfLine <- <(('\r' '\n') / '\n' / '\r')> */
		func() bool {
			position255, tokenIndex255 := position, tokenIndex
			{
				position256 := position
				{
					position257, tokenIndex257 := position, tokenIndex
					if buffer[position] != rune('\r') {
						goto l258
					}
					position++
					if buffer[position] != rune('\n') {
						goto l258
					}
					position++
					goto l257
				l258:
					position, tokenIndex = position257, tokenIndex257
					if buffer[position] != rune('\n') {
						goto l259
					}
					position++
					goto l257
				l259:
					position, tokenIndex = position257, tokenIndex257
					if buffer[position] != rune('\r') {
						goto l255
					}
					position++
				}
			l257:
				add(ruleEndOfLine, position256)
			}
			return true
		l255:
			position, tokenIndex = position255, tokenIndex255
			return false
		},
		/* 29 EndOfFile <- <!.> */
		nil,
		nil,
		/* 32 Action0 <- <{ p.AddVarIdentifier(text) }> */
		nil,
		/* 33 Action1 <- <{ p.LineDone() }> */
		nil,
		/* 34 Action2 <- <{ p.AddDeclarationIdentifier(text) }> */
		nil,
		/* 35 Action3 <- <{ p.AddAction(text) }> */
		nil,
		/* 36 Action4 <- <{ p.AddEntity(text) }> */
		nil,
		/* 37 Action5 <- <{ p.LineDone() }> */
		nil,
		/* 38 Action6 <- <{ p.AddParamKey(text) }> */
		nil,
		/* 39 Action7 <- <{  p.AddParamHoleValue(text) }> */
		nil,
		/* 40 Action8 <- <{  p.AddParamAliasValue(text) }> */
		nil,
		/* 41 Action9 <- <{  p.AddParamRefValue(text) }> */
		nil,
		/* 42 Action10 <- <{ p.AddParamCidrValue(text) }> */
		nil,
		/* 43 Action11 <- <{ p.AddParamIpValue(text) }> */
		nil,
		/* 44 Action12 <- <{ p.AddParamValue(text) }> */
		nil,
		/* 45 Action13 <- <{ p.AddParamIntValue(text) }> */
		nil,
		/* 46 Action14 <- <{ p.AddParamValue(text) }> */
		nil,
		/* 47 Action15 <- <{  p.AddVarHoleValue(text) }> */
		nil,
		/* 48 Action16 <- <{ p.AddVarCidrValue(text) }> */
		nil,
		/* 49 Action17 <- <{ p.AddVarIpValue(text) }> */
		nil,
		/* 50 Action18 <- <{ p.AddVarValue(text) }> */
		nil,
		/* 51 Action19 <- <{ p.AddVarIntValue(text) }> */
		nil,
		/* 52 Action20 <- <{ p.AddVarValue(text) }> */
		nil,
		/* 53 Action21 <- <{ p.LineDone() }> */
		nil,
	}
	p.rules = _rules
}
