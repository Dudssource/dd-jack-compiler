package compiler

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"strings"
)

type compilationEngine struct {
	tknzr          *jackTokenizer
	dstFile        io.Writer
	builder        *strings.Builder
	errorList      []error
	isDebugEnabled bool
}

func newCompilationEngine(tknzr *jackTokenizer, dstFile io.Writer) *compilationEngine {
	return &compilationEngine{
		tknzr:          tknzr,
		dstFile:        dstFile,
		builder:        &strings.Builder{},
		isDebugEnabled: strings.EqualFold(os.Getenv("JACK_COMPILER_DEBUG"), "true"),
	}
}

func (ce *compilationEngine) debug(format string, v ...any) {
	if !ce.isDebugEnabled {
		return
	}
	_, file, no, ok := runtime.Caller(1)
	if ok {
		format = fmt.Sprintf("%s:%d %s", file, no, format)
	}
	log.Printf(format, v...)
}

func (ce *compilationEngine) compile() error {

	// compile class
	ce.compileClass()

	// encode xml with indentation
	decoder := xml.NewDecoder(strings.NewReader(ce.builder.String()))
	encoder := xml.NewEncoder(ce.dstFile)
	encoder.Indent("", "  ")

	for {
		tokenXml, err := decoder.RawToken()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			ce.errorList = append(ce.errorList, err)
			break
		}

		encoder.EncodeToken(tokenXml)
	}

	if err := encoder.Close(); err != nil {
		ce.errorList = append(ce.errorList, err)
	}

	return errors.Join(ce.errorList...)
}

func (ce *compilationEngine) compileClass() {
	// start token
	ce.tknzr.advance()
	ce.print("<class>")
	{
		ce.process("class")
		ce.process("className")
		ce.process("{")
		{
			// classVarDec*
			for ce.tknzr.token().lex == Keyword {
				switch ce.tknzr.token().value {
				case "static", "field":
					ce.print("<classVarDec>")
					ce.process(ce.tknzr.token().value) // static, field
					ce.process("type")
					ce.process("varName")
					// (','varName)*
					for ce.tknzr.token().value == "," {
						ce.process(",")
						ce.process("varName")
					}
					ce.process(";")
					ce.print("</classVarDec>")
					continue
				}
				break // for
			}

			// subroutineDec*
			for ce.tknzr.token().lex == Keyword {
				switch ce.tknzr.token().value {
				case "constructor", "function", "method":
					ce.print("<subroutineDec>")
					ce.process(ce.tknzr.token().value) // "constructor", "function", "method"
					ce.process("void|type")
					ce.process("subroutineName")
					ce.process("(")
					ce.compileParameterList()
					ce.process(")")
					ce.compileSubRoutineBody()
					ce.print("</subroutineDec>")
					continue
				}
				break // for
			}
		}
		ce.process("}")
	}
	ce.print("</class>")
}

func (ce *compilationEngine) compileParameterList() {
	ce.print("<parameterList>")
	{
		if ce.tknzr.token().value != ")" {
			ce.process("type")
			ce.process("varName")
			for ce.tknzr.token().value == "," {
				ce.process(",")
				ce.process("type")
				ce.process("varName")
			}
		} else {
			ce.print("\n")
		}
	}
	ce.print("</parameterList>")
}

func (ce *compilationEngine) compileSubRoutineBody() {
	ce.print("<subroutineBody>")
	{
		ce.process("{")
		{
			for ce.tknzr.token().value == "var" {
				ce.print("<varDec>")
				ce.process("var")
				ce.process("type")
				ce.process("varName")
				for ce.tknzr.token().value == "," {
					ce.process(",")
					ce.process("varName")
				}
				ce.process(";")
				ce.print("</varDec>")
			}
			ce.compileStatements()
		}
		ce.process("}")
	}
	ce.print("</subroutineBody>")
}

func (ce *compilationEngine) compileStatements() {
	ce.print("<statements>")
	{
		for ce.tknzr.hasMoreTokens() {
			switch ce.tknzr.token().value {
			case "while":
				ce.compileWhile()
				continue // for
			case "let":
				ce.compileLet()
				continue // for
			case "if":
				ce.compileIf()
				continue // for
			case "do":
				ce.compileDo()
				continue // for
			case "return":
				ce.compileReturn()
				continue // for
			}
			break // for
		}
	}
	ce.print("</statements>")
}

func (ce *compilationEngine) compileLet() {
	ce.print("<letStatement>")
	{
		ce.process("let")
		ce.process("varName")
		if ce.tknzr.token().value == "[" {
			ce.process("[")
			ce.compileExpression()
			ce.process("]")
		}
		ce.process("=")
		ce.compileExpression()
		ce.process(";")
	}
	ce.print("</letStatement>")
}

func (ce *compilationEngine) compileReturn() {
	ce.print("<returnStatement>")
	ce.process("return")
	// expression?
	if ce.tknzr.token().value != ";" {
		ce.compileExpression()
	}
	ce.process(";")
	ce.print("</returnStatement>")
}

func (ce *compilationEngine) compileIf() {

	ce.print("<ifStatement>")
	{
		ce.process("if")
		ce.process("(")
		{
			ce.compileExpression()
		}
		ce.process(")")
		ce.process("{")
		{
			ce.compileStatements()
		}
		ce.process("}")
	}
	{
		// ('else''{statements'}')?
		if ce.tknzr.token().value == "else" {
			ce.process("else")
			ce.process("{")
			{
				ce.compileStatements()
			}
			ce.process("}")
		}
	}
	ce.print("</ifStatement>")
}

func (ce *compilationEngine) compileWhile() {

	ce.print("<whileStatement>")
	{
		ce.process("while")
		ce.process("(")
		{
			ce.compileExpression()
		}
		ce.process(")")
		ce.process("{")
		{
			ce.compileStatements()
		}
		ce.process("}")
	}
	ce.print("</whileStatement>")
}

func (ce *compilationEngine) compileDo() {
	ce.print("<doStatement>")
	{
		ce.process("do")
		// subroutineCall
		{
			// term (without tag)
			ce.compileTerm(false)

			// optional (op term)*
			for ce.tknzr.token().lex == Symbol {
				switch ce.tknzr.token().value {
				// op
				case "+", "-", "=", ">", "<", "*", "/", "&", "|":
					ce.printXML(ce.tknzr.token())
					ce.tknzr.advance()
					ce.compileTerm(true)
					continue
				}
				break // for
			}
		}
		ce.process(";")
	}
	ce.print("</doStatement>")
}

func (ce *compilationEngine) compileExpression() {
	ce.print("<expression>")
	{
		// term
		ce.compileTerm(true)

		// optional (op term)*
		for ce.tknzr.token().lex == Symbol {
			switch ce.tknzr.token().value {
			// op
			case "+", "-", "=", ">", "<", "*", "/", "&", "|":
				ce.printXML(ce.tknzr.token())
				ce.tknzr.advance()
				ce.compileTerm(true)
				continue
			}
			break // for
		}
	}
	ce.print("</expression>")
}

func (ce *compilationEngine) compileTerm(tag bool) {
	if tag {
		ce.print("<term>")
	}
	{
		switch ce.tknzr.token().lex {
		case Identifier:
			ce.process("identifier") // varName, className, subroutineName
			switch ce.tknzr.token().value {
			case "[":
				ce.process("[")
				ce.compileExpression()
				ce.process("]")
			case "(":
				ce.process("(")
				ce.compileExpressionList()
				ce.process(")")
			case ".":
				ce.process(".")
				ce.process("subroutineName")
				ce.process("(")
				ce.compileExpressionList()
				ce.process(")")
			}
		case IntConst, StringConst:
			ce.process("constant")
		case Keyword: // keyword constant
			switch ce.tknzr.token().value {
			case "true", "false", "null", "this":
				ce.process(ce.tknzr.token().value)
			default:
				ce.expected("keywordConstant")
			}
		case Symbol:
			// (expression)
			if ce.tknzr.token().value == "(" {
				ce.process("(")
				ce.compileExpression()
				ce.process(")")

				// unaryOp
			} else if ce.tknzr.token().value == "-" || ce.tknzr.token().value == "~" {
				ce.process(ce.tknzr.token().value) // unaryOp - ~
				ce.compileTerm(true)
			} else {
				ce.expected("parenthesis or unaryOp")
			}
		default:
			ce.expected("varName or constant")
		}
	}
	if tag {
		ce.print("</term>")
	}
}

func (ce *compilationEngine) compileExpressionList() {
	ce.print("<expressionList>")
	{
		// handle empty expression list
		if ce.tknzr.token().value != ")" {
			ce.compileExpression()
			// (, expression)*
			for ce.tknzr.token().value == "," {
				ce.process(",")
				ce.compileExpression()
			}
		} else {
			ce.print("\n")
		}
	}
	ce.print("</expressionList>")
}

func (ce *compilationEngine) expected(expected any) {
	err := fmt.Errorf("syntax error : expected %v, got %s", expected, ce.tknzr.token().String())
	ce.errorList = append(ce.errorList, err)
}

func (ce *compilationEngine) processType() {
	if ce.tknzr.token().lex == Keyword {
		if ce.tknzr.token().value == "int" ||
			ce.tknzr.token().value == "boolean" || ce.tknzr.token().value == "char" {
			// type
			ce.printXML(ce.tknzr.token())
		} else {
			ce.expected("type : int, boolean or char")
		}
	} else if ce.tknzr.token().lex == Identifier {
		// className
		ce.printXML(ce.tknzr.token())
	} else {
		ce.expected("type or className")
	}
}

func (ce *compilationEngine) process(expected string) {
	ce.debug(expected)
	switch expected {
	case "varName", "className", "subroutineName", "identifier":
		if ce.tknzr.token().lex == Identifier {
			ce.printXML(ce.tknzr.token())
		} else {
			ce.expected(expected)
		}
	case "constant":
		if ce.tknzr.token().lex == StringConst || ce.tknzr.token().lex == IntConst {
			ce.printXML(ce.tknzr.token())
		} else {
			ce.expected(expected)
		}
	case "constructor", "function", "method":
		if ce.tknzr.token().lex == Keyword {
			ce.printXML(ce.tknzr.token())
		} else {
			ce.expected(expected)
		}

	case "void|type":
		if ce.tknzr.token().lex == Keyword && ce.tknzr.token().value == "void" {
			ce.printXML(ce.tknzr.token())
		} else {
			ce.processType()
		}
	case "type":
		ce.processType()
	default:
		if expected == ce.tknzr.token().value {
			ce.printXML(ce.tknzr.token())
		} else {
			ce.expected(expected)
		}
	}

	// advance tokenizer
	ce.tknzr.advance()
}

func (ce *compilationEngine) print(value string) {
	_, _ = ce.builder.WriteString(value)
}

func (ce *compilationEngine) printXML(token *Token) {
	ce.print(fmt.Sprintf("<%s> ", token.lex))
	_ = xml.EscapeText(ce.builder, []byte(token.value))
	ce.print(fmt.Sprintf(" </%s>", token.lex))
}
