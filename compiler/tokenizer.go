package compiler

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"unicode"
)

type TokenType string

func (t TokenType) String() string {
	return string(t)
}

const (
	Keyword     TokenType = "keyword"
	Symbol      TokenType = "symbol"
	IntConst    TokenType = "integerConstant"
	StringConst TokenType = "stringConstant"
	Identifier  TokenType = "identifier"
)

var (
	keywordRegex    = regexp.MustCompile(`^(class|constructor|function|field|method|static|var|int|char|boolean|void|true|false|null|this|let|do|if|else|while|return)$`)
	identifierRegex = regexp.MustCompile(`^[a-zA-Z0-9_]{1}$`)
)

type Token struct {
	lex    TokenType
	value  string
	lineNo int
}

func (t *Token) String() string {
	return fmt.Sprintf("lex=%s, value=%s, lineNo=%d", t.lex, t.value, t.lineNo)
}

func newToken(symbol TokenType, value string, lineNo int) *Token {
	return &Token{lex: symbol, lineNo: lineNo, value: value}
}

type jackTokenizer struct {
	reader       *bufio.Reader
	lineNo       int
	more         bool
	currentToken *Token
}

func newTokenizer(srcFile io.Reader) *jackTokenizer {
	return &jackTokenizer{
		lineNo: 1,
		reader: bufio.NewReader(srcFile),
	}
}

// readChar low level function to read the next character on the stream of characters
func (tkn *jackTokenizer) readChar() (rune, bool) {
	// current char
	ch, _, err := tkn.reader.ReadRune()
	if err != nil {
		if errors.Is(err, io.EOF) {
			return unicode.MaxRune, false
		}
		panic(err)
	}
	if ch == '\n' {
		tkn.lineNo++
	}
	return ch, true
}

// rewind low level function to rewind the stream of characters (not the token)
func (tkn *jackTokenizer) rewind() {
	_ = tkn.reader.UnreadRune()
}

// token returns the current token or nil
func (tkn *jackTokenizer) token() *Token {
	return tkn.currentToken
}

// hasMoreTokens returns if more tokens are available
func (tkn *jackTokenizer) hasMoreTokens() bool {
	return tkn.more && tkn.currentToken != nil
}

// advance advances the tokenizer
func (tkn *jackTokenizer) advance() {
	tkn.currentToken, tkn.more = tkn.getNextToken()
}

// getNextToken low level access to token and next
func (tkn *jackTokenizer) getNextToken() (*Token, bool) {

	for ch, hasNext := tkn.readChar(); hasNext; ch, hasNext = tkn.readChar() {

		switch ch {

		case '/': // symbol / or comment // or multi-line comment /* */
			if sch, hasNext := tkn.readChar(); hasNext {
				if sch == '/' { // is a comment, ignore the rest of line
					for sch, hasNext := tkn.readChar(); hasNext; sch, hasNext = tkn.readChar() {
						if sch == '\n' || sch == '\r' {
							// rewind
							tkn.rewind()
							break
						}
					}
					continue
				}
				multiLine := false
				if sch == '*' {
					multiLine = true
					lastChar := false
					for sch, hasNext := tkn.readChar(); hasNext; sch, hasNext = tkn.readChar() {
						if lastChar && sch == '/' {
							break
						}
						if sch == '*' {
							lastChar = true
						} else {
							lastChar = false
						}
					}
				}
				// skip everything up here
				if multiLine {
					continue
				}
				// rewind
				tkn.rewind()
				// is symbol
				return newToken(Symbol, string(ch), tkn.lineNo), true
			}

		case '{', '}', '[', ']', '(', ')', ',', '.', ';', '+', '*', '-', '&', '|', '<', '>', '=', '~': // symbols (except /)
			return newToken(Symbol, string(ch), tkn.lineNo), true

		case '\n', '\r': // line break
			continue

		case '"': // string constant
			var sb strings.Builder
			// read while string
			for sch, hasNext := tkn.readChar(); hasNext; sch, hasNext = tkn.readChar() {
				// end of string
				if sch == '"' {
					return newToken(StringConst, sb.String(), tkn.lineNo), true
				}
				sb.WriteRune(sch)
			}

		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9': // integer constant
			var sb strings.Builder
			sb.WriteRune(ch)
			// read while digit
			for sch, hasNext := tkn.readChar(); hasNext; sch, hasNext = tkn.readChar() {
				if unicode.IsDigit(sch) {
					sb.WriteRune(sch)
				} else {
					// rewind
					tkn.rewind()
					break
				}
			}
			return newToken(IntConst, sb.String(), tkn.lineNo), true

		default:

			// ignore white spaces, tabs, etc
			if unicode.IsSpace(ch) {
				continue
			}

			// identifier or keyword
			if identifierRegex.MatchString(string(ch)) {

				var sb strings.Builder
				sb.WriteRune(ch)
				// read while letter or _ or digit
				for sch, hasNext := tkn.readChar(); hasNext; sch, hasNext = tkn.readChar() {
					if identifierRegex.MatchString(string(sch)) {
						sb.WriteRune(sch)
					} else {
						// rewind
						tkn.rewind()
						break
					}
				}

				token := newToken(Identifier, sb.String(), tkn.lineNo)

				// check for keyword
				if keywordRegex.MatchString(token.value) {
					token.lex = Keyword
				}

				return token, true
			}
		}
	}

	return nil, false
}
