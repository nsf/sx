package sx

import (
	"errors"
)

//----------------------------------------------------------------------------
// ast node
//----------------------------------------------------------------------------

type Node struct {
	List  []Node
	Value string
}

//----------------------------------------------------------------------------
// parser
//----------------------------------------------------------------------------

func isSpace(b int) bool {
	switch b {
	case ' ', '\t', '\n', '\r':
		return true
	}
	return false
}

func isScalar(b int) bool {
	switch b {
	case ' ', '\t', '\n', '\r', '"', '`', '(', ')', ';':
		return false
	}
	return true
}

func isHex(b int) (int, bool) {
	switch {
	case '0' <= b && b <= '9':
		return b - '0', true
	case 'a' <= b && b <= 'f':
		return 10 + b - 'a', true
	case 'A' <= b && b <= 'F':
		return 10 + b - 'A', true
	}
	return eof, false
}

const eof int = -1

type parser struct {
	data []byte
	ptr  int // pointer into 'data'
	err  error
}

func (p *parser) error(msg string) int {
	p.ptr = len(p.data)
	p.err = errors.New(msg)
	return eof
}

// current byte or EOF
func (p *parser) current() int {
	if p.ptr == len(p.data) {
		return eof
	}
	return int(p.data[p.ptr])
}

// next Nth byte or EOF
func (p *parser) next(n int) int {
	if p.ptr+n >= len(p.data) {
		return eof
	}
	return int(p.data[p.ptr])
}

// increment pointer and return current byte or EOF
func (p *parser) advance() int {
	if p.ptr < len(p.data) {
		p.ptr++
	}
	return p.current()
}

func (p *parser) advanceN(n int) int {
	p.ptr += n
	if p.ptr > len(p.data) {
		p.ptr = len(p.data)
	}
	return p.current()
}

// unread length
func (p *parser) unreadLen() int {
	return len(p.data) - p.ptr
}

func (p *parser) skipToNonSpace() {
	for b := p.current(); b != eof && isSpace(b); b = p.advance() {
	}
}

func (p *parser) skipComment() {
	for b := p.current(); b != eof && b != '\n'; b = p.advance() {
	}
}

func (p *parser) matches(s string) bool {
	if p.unreadLen() < len(s) {
		return false
	}

	ptr := p.ptr
	for i := 0; i < len(s); i++ {
		if p.data[ptr+i] != s[i] {
			return false
		}
	}
	return true
}

func (p *parser) parseScalar() Node {
	buf := []byte{}
	for b := p.current(); b != eof && isScalar(b); b = p.advance() {
		buf = append(buf, byte(b))
	}
	return Node{Value: string(buf)}
}

// All known escape sequences are single bytes.
// Sets an error and returns 'eof' on invalid escape sequence.
//
// Expects pointer at opening `\`, leaves pointer at the last character of
// escape sequence.
func (p *parser) parseEscapeSequence() int {
	b := p.advance() // step into the literal from `\`
	switch b {
	case '"':
		return '"'
	case '\\':
		return '\\'
	case 'r':
		return '\r'
	case 'n':
		return '\n'
	case 't':
		return '\t'
	case 'x':
		// raw byte: \xFF
		if p.unreadLen() < 3 {
			return p.error("unexpected eof when parsing a string escape sequence (hex literal)")
		}
		a, ok := isHex(int(p.data[p.ptr+1]))
		if !ok {
			return p.error("invalid first hex digit in string escape sequence")
		}
		b, ok := isHex(int(p.data[p.ptr+2]))
		if !ok {
			return p.error("invalid second hex digit in string escape sequence")
		}
		p.advanceN(2) // put pointer to the last character of sequence
		return a*16 + b
	default:
		if b == eof {
			return p.error("unexpected eof when parsing a string escape sequence")
		} else {
			return p.error("invalid escape sequence")
		}
	}
}

// Expects pointer at opening `"`, leaves pointer at the next character after
// closing `"`.
func (p *parser) parseStringLiteral() (Node, bool) {
	buf := []byte{}
	for b := p.advance(); b != eof; b = p.advance() {
		switch b {
		case '\\':
			b = p.parseEscapeSequence()
			if b == eof {
				return Node{}, false
			}
			fallthrough
		default:
			buf = append(buf, byte(b))
		case '"':
			p.advance()
			return Node{Value: string(buf)}, true
		case '\n':
			p.error(`unexpected '\n' in a string literal, allowed in multi-line strings only`)
			return Node{}, false
		}
	}
	p.error(`unexpected eof, missing terminating '"' in a string literal`)
	return Node{}, false
}

// Expects pointer at opening '`', leaves pointer at the next character after
// closing '`'.
func (p *parser) parseRawStringLiteral() (Node, bool) {
	buf := []byte{}
	for b := p.advance(); b != eof; b = p.advance() {
		switch b {
		case '`':
			p.advance()
			return Node{Value: string(buf)}, true
		case '\n':
			p.error(`unexpected '\n' in a raw string literal, allowed in multi-line strings only`)
			return Node{}, false
		default:
			buf = append(buf, byte(b))
		}
	}
	p.error("unexpected eof, missing terminating '`' in a raw string literal")
	return Node{}, false
}

// Expects pointer at opening '|', leaves pointer at unexpected EOF or '\n'.
func (p *parser) parseRawLine(buf []byte) []byte {
	if p.advance() == ' ' {
		p.advance()
	}
	for b := p.current(); b != eof && b != '\n'; b = p.advance() {
		switch b {
		case '\r':
			// skip '\r'
		default:
			buf = append(buf, byte(b))
		}
	}
	return buf
}

// Expects pointer at opening '`', assuming that the sequence is '`\n' or
// '`\r\n'. Leaves pointer at the next character after cloing '`'.
func (p *parser) parseMultiLineStringLiteral() (Node, bool) {
	if p.next(1) == '\r' {
		p.advanceN(3)
	} else {
		p.advanceN(2)
	}

	buf := []byte{}
	for {
		p.skipToNonSpace()
		switch p.current() {
		case '`':
			p.advance()
			return Node{Value: string(buf)}, true
		case '|':
			if len(buf) != 0 {
				buf = append(buf, '\n')
			}
			buf = p.parseRawLine(buf)
			p.advance()
		case eof:
			p.error("unexpected eof when parsing a multi-line string literal")
			return Node{}, false
		default:
			p.error("invalid beginning of a string in a multi-line string literal, '`' or '|' expected")
			return Node{}, false
		}
	}
}

// Expects pointer at opening '(', leaves pointer at the next character after
// closing ')'.
func (p *parser) parseList() (Node, bool) {
	out := []Node{}
	p.advance() // skip opening '('
	for {
		p.skipToNonSpace()
		switch p.current() {
		case eof:
			p.error("unexpected eof when parsing a list")
			return Node{}, false
		case ')':
			p.advance()
			return Node{List: out}, true
		default:
			node, ok := p.parseSingleNode()
			if !ok {
				return Node{}, false
			}
			out = append(out, node)
		}
	}
}

func (p *parser) parseSingleNode() (Node, bool) {
	for {
		p.skipToNonSpace()
		switch p.current() {
		case '(':
			return p.parseList()
		case ')':
			p.error("unmatched closing parenthesis ')'")
			return Node{}, false
		case '"':
			return p.parseStringLiteral()
		case '`':
			if p.matches("`\n") || p.matches("`\r\n") {
				return p.parseMultiLineStringLiteral()
			} else {
				return p.parseRawStringLiteral()
			}
		case ';':
			p.skipComment()
		default:
			return p.parseScalar(), true
		case eof:
			return Node{}, false
		}
	}
}

func (p *parser) parse() []Node {
	var out []Node
	for {
		node, ok := p.parseSingleNode()
		if !ok {
			break
		}
		out = append(out, node)
	}
	if p.err == nil {
		return out
	}
	return nil
}

func Parse(data []byte) ([]Node, error) {
	p := parser{data: data}
	ast := p.parse()
	return ast, p.err
}
