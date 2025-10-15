package compiler

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

var jackOSAPI = map[string]bool{
	"Math":     true,
	"String":   true,
	"Array":    true,
	"Output":   true,
	"Screen":   true,
	"Keyboard": true,
	"Memory":   true,
	"Sys":      true,
}

type compilationEngine struct {
	tknzr          *jackTokenizer
	builder        *strings.Builder
	errorList      []error
	symbolTable    *symbolTable
	className      string
	writer         *vmWriter
	isDebugEnabled bool
	labelsCounter  int
}

func newCompilationEngine(tknzr *jackTokenizer, dstFile io.Writer) *compilationEngine {
	return &compilationEngine{
		tknzr:          tknzr,
		builder:        &strings.Builder{},
		symbolTable:    newSymbolTable(),
		writer:         &vmWriter{dstFile: dstFile},
		isDebugEnabled: strings.EqualFold(os.Getenv("JACK_COMPILER_DEBUG"), "true"),
	}
}

func (ce *compilationEngine) compile() error {

	// compile class
	ce.compileClass()

	return errors.Join(ce.errorList...)
}

func (ce *compilationEngine) compileClass() {
	// start token
	ce.tknzr.advance()

	// <class>
	ce.check("class")
	{
		// save class name
		ce.className = ce.tokenValue()
		ce.check("className")
		ce.check("{")
		{
			// classVarDec*
			for ce.tknzr.token().lex == Keyword {

				var (
					kind, name, ttype string
				)

				switch ce.tokenValue() {
				case "static", "field":
					// <classVarDec>

					// kind
					kind = ce.tokenValue()
					ce.check(ce.tokenValue()) // static, field

					// type
					ttype = ce.tokenValue()
					ce.check("type")

					// varName
					name = ce.tokenValue()
					ce.check("varName")

					// add to class symbol level (0)
					ce.symbolTable.define(name, ttype, kind)

					// (','varName)*
					for ce.tokenValue() == "," {
						ce.check(",")

						// varName
						name = ce.tokenValue()
						ce.check("varName")

						// add to class symbol level (0)
						ce.symbolTable.define(name, ttype, kind)
					}
					ce.check(";")
					// </classVarDec>
					continue
				}
				break // for
			}

			if ce.isDebugEnabled {
				ce.symbolTable.debug()
			}

			// subroutineDec*
			for ce.tknzr.token().lex == Keyword {
				switch ce.tokenValue() {
				case "constructor", "function", "method":

					subroutineType := ce.tokenValue()

					// <subroutineDec>

					// next level
					ce.symbolTable.next()

					// "constructor", "function", "method"
					ce.symbolTable.define("this", ce.className, "argument")
					ce.check(ce.tokenValue())

					if ce.tokenValue() == "void" {
						ce.check("void")
					} else {
						ce.check("type")
					}

					subroutineName := fmt.Sprintf("%s.%s", ce.className, ce.tokenValue())
					ce.check("subroutineName")
					ce.check("(")
					ce.compileParameterList()
					ce.check(")")

					// write function
					switch subroutineType {
					case "function":
						ce.writer.writeFunction(subroutineName, ce.symbolTable.varCount("argument")-1)
					case "method":
						ce.writer.writeFunction(subroutineName, ce.symbolTable.varCount("argument"))
						ce.writer.writePush("argument", 0)
						ce.writer.writePop("pointer", 0)
					case "constructor":
						ce.writer.writeFunction(subroutineName, 0)
						ce.writer.writePush("constant", ce.symbolTable.varCount("this"))
						ce.writer.writeCall("Memory.alloc", 1)
						ce.writer.writePop("pointer", 0)
					}

					ce.compileSubRoutineBody()

					if ce.isDebugEnabled {
						ce.symbolTable.debug()
					}

					// previous level
					ce.symbolTable.prev()

					// </subroutineDec>
					continue
				}
				break // for
			}
		}
		ce.check("}")

		if ce.isDebugEnabled {
			ce.symbolTable.debug()
		}
	}
	// </class>
}

func (ce *compilationEngine) compileParameterList() {
	// <parameterList>
	{
		if ce.tokenValue() != ")" {

			var (
				name, ttype string
			)

			// type
			ttype = ce.tokenValue()
			ce.check("type")

			// varName
			name = ce.tokenValue()
			ce.check("varName")

			// add to symbol table
			ce.symbolTable.define(name, ttype, "argument")

			for ce.tokenValue() == "," {
				ce.check(",")

				// type
				ttype = ce.tokenValue()
				ce.check("type")

				// varName
				name = ce.tokenValue()
				ce.check("varName")

				// add to symbol table
				ce.symbolTable.define(name, ttype, "argument")
			}
		}
	}
	// </parameterList>
}

func (ce *compilationEngine) compileSubRoutineBody() {
	// <subroutineBody>
	{
		ce.check("{")
		{
			for ce.tokenValue() == "var" {

				var (
					name, ttype string
				)

				// <varDec>
				ce.check("var")

				// type
				ttype = ce.tokenValue()
				ce.check("type")

				// varName
				name = ce.tokenValue()
				ce.check("varName")

				// add to symbol table
				ce.symbolTable.define(name, ttype, "local")

				for ce.tokenValue() == "," {
					ce.check(",")

					// varName
					name = ce.tokenValue()
					ce.check("varName")

					// add to symbol table
					ce.symbolTable.define(name, ttype, "local")
				}

				ce.check(";")
				// </varDec>
			}
			ce.compileStatements()
		}
		ce.check("}")
	}
	// </subroutineBody>
}

func (ce *compilationEngine) compileStatements() {
	// <statements>
	{
		for ce.tknzr.hasMoreTokens() {
			switch ce.tokenValue() {
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
	// </statements>
}

func (ce *compilationEngine) compileLet() {
	// <letStatement>
	{
		ce.check("let")
		varName := ce.tknzr.token()
		ce.check("varName")
		if ce.tokenValue() == "[" {
			ce.check("[")
			ce.compileExpression()
			ce.check("]")
		}
		ce.check("=")
		ce.compileExpression()
		ce.check(";")

		if item, ok := ce.symbolTable.find(varName.value); ok {
			// pop
			ce.writer.writePop(item.kind, item.position)
		} else {
			ce.undeclared(varName)
		}
	}
	// </letStatement>
}

func (ce *compilationEngine) compileReturn() {
	// <returnStatement>
	ce.check("return")
	// expression?
	if ce.tokenValue() != ";" {
		ce.compileExpression()
	} else {
		// no return variable
		ce.writer.writePush("constant", 0)
	}
	ce.check(";")
	// return
	ce.writer.writeReturn()
	// </returnStatement>
}

func (ce *compilationEngine) label() string {
	value := strconv.Itoa(ce.labelsCounter)
	ce.labelsCounter++
	return fmt.Sprintf("%s_%s", ce.className, value)
}

func (ce *compilationEngine) compileIf() {

	var (
		labelA = ce.label()
		labelB = ce.label()
	)

	// <ifStatement>
	{
		ce.check("if")
		ce.check("(")
		{
			ce.compileExpression()
			// not
			ce.writer.writeUnaryOp("~")
			// if-goto label A
			ce.writer.writeIf(labelA)
		}
		ce.check(")")
		ce.check("{")
		{
			ce.compileStatements()
		}
		ce.check("}")
	}
	{
		// ('else''{statements'}')?
		if ce.tokenValue() == "else" {

			// goto label B
			ce.writer.writeGoto(labelB)

			ce.check("else")
			ce.check("{")
			{
				// label A
				ce.writer.writeLabel(labelA)
				ce.compileStatements()
				// label B
				ce.writer.writeLabel(labelB)
			}
			ce.check("}")
		} else {
			// label A
			ce.writer.writeLabel(labelA)
		}
	}
	// </ifStatement>
}

func (ce *compilationEngine) compileWhile() {

	var (
		labelA = ce.label()
		labelB = ce.label()
	)

	// <whileStatement>
	{
		// label A
		ce.writer.writeLabel(labelA)

		ce.check("while")
		ce.check("(")
		{
			ce.compileExpression()
			// not
			ce.writer.writeUnaryOp("~")
			// if-goto LB
			ce.writer.writeIf(labelB)
		}
		ce.check(")")
		ce.check("{")
		{
			ce.compileStatements()
			// goto LA
			ce.writer.writeGoto(labelA)
		}
		ce.check("}")

		// label B
		ce.writer.writeLabel(labelB)

	}
	// </whileStatement>
}

func (ce *compilationEngine) compileDo() {
	// <doStatement>
	{
		ce.check("do")
		// subroutineCall
		{
			// term (without tag)
			ce.compileTerm()

			// optional (op term)*
			for ce.tknzr.token().lex == Symbol {
				switch ce.tokenValue() {
				// op
				case "+", "-", "=", ">", "<", "*", "/", "&", "|":
					//ce.printXML(ce.tknzr.token())
					ce.tknzr.advance()
					ce.compileTerm()
					continue
				}
				break // for
			}
		}

		// ignore returned value
		ce.writer.writePop("temp", 0)

		ce.check(";")
	}
	// </doStatement>
}

func (ce *compilationEngine) compileExpression() {
	// <expression>
	{
		// term
		ce.compileTerm()

		// optional (op term)*
		for ce.tknzr.token().lex == Symbol {
			switch ce.tokenValue() {
			// op
			case "+", "-", "=", ">", "<", "*", "/", "&", "|":
				op := ce.tokenValue()
				ce.tknzr.advance()
				ce.compileTerm()
				ce.writer.writeOp(op)
				continue
			}
			break // for
		}
	}

	// </expression>
}

func (ce *compilationEngine) compileTerm() {

	// <term>
	{
		switch ce.tknzr.token().lex {
		case Identifier:
			// save varName
			identifier := ce.tknzr.token()
			ce.check("identifier") // varName, className, subroutineName
			switch ce.tokenValue() {
			case "[":
				ce.check("[")
				ce.compileExpression()
				ce.check("]")
			case "(":

				target := ce.className
				// push var
				if objectTbl, ok := ce.symbolTable.find("this"); ok {
					target = objectTbl.ttype
					ce.writer.writePush(objectTbl.kind, objectTbl.position)
				} else {
					ce.undeclared(identifier)
				}
				methodName := fmt.Sprintf("%s.%s", target, identifier.value)

				ce.check("(")
				expN := ce.compileExpressionList()
				ce.check(")")
				// call f n
				ce.writer.writeCall(methodName, expN+1)
			case ".":
				ce.check(".")
				padding := 0
				target := identifier.value
				// push var
				if objectTbl, ok := ce.symbolTable.find(identifier.value); ok {
					padding = 1
					target = objectTbl.ttype
					ce.writer.writePush(objectTbl.kind, objectTbl.position)
				} else if !jackOSAPI[target] {
					ce.undeclared(identifier)
				}
				subroutineName := fmt.Sprintf("%s.%s", target, ce.tokenValue())
				ce.check("subroutineName")
				ce.check("(")
				expN := ce.compileExpressionList()
				ce.check(")")
				// call f n
				ce.writer.writeCall(subroutineName, expN+padding)
			default:
				// push var
				if varTbl, ok := ce.symbolTable.find(identifier.value); ok {
					ce.writer.writePush(varTbl.kind, varTbl.position)
				} else {
					ce.undeclared(identifier)
				}
			}
		case IntConst:
			val, _ := strconv.Atoi(ce.tokenValue())
			ce.writer.writePush("constant", val)
			ce.check("constant")
		case StringConst:
			ce.check("constant")
			// TODO
		case Keyword: // keyword constant
			switch ce.tokenValue() {
			case "true":
				ce.check(ce.tokenValue())
				// true
				ce.writer.writePush("constant", 1)
			case "false":
				ce.check(ce.tokenValue())
				// false
				ce.writer.writePush("constant", -1)
			case "null", "this":
				ce.check(ce.tokenValue())
			default:
				ce.expected("keywordConstant")
			}
		case Symbol:

			// (expression)
			if ce.tokenValue() == "(" {
				ce.check("(")
				ce.compileExpression()
				ce.check(")")

				// unaryOp
			} else if ce.tokenValue() == "-" || ce.tokenValue() == "~" {
				// unary op
				unaryOp := ce.tokenValue()
				ce.check(ce.tokenValue()) // unaryOp - ~
				ce.compileTerm()
				// unary op
				ce.writer.writeUnaryOp(unaryOp)
			} else {
				ce.expected("parenthesis or unaryOp")
			}
		default:
			ce.expected("varName or constant")
		}
	}

	// </term>
}

func (ce *compilationEngine) compileExpressionList() int {
	// <expressionList>
	cnt := 0
	{
		// handle empty expression list
		if ce.tokenValue() != ")" {
			cnt++
			ce.compileExpression()
			// (, expression)*
			for ce.tokenValue() == "," {
				ce.check(",")
				cnt++
				ce.compileExpression()
			}
		}
	}
	return cnt
	// </expressionList>
}

func (ce *compilationEngine) expected(expected any) {
	err := fmt.Errorf("syntax error : expected %v, got %s", expected, ce.tknzr.token().String())
	ce.errorList = append(ce.errorList, err)
}

func (ce *compilationEngine) undeclared(tkn *Token) {
	err := fmt.Errorf("compiler error : undeclared var %s", tkn.String())
	ce.errorList = append(ce.errorList, err)
}

func (ce *compilationEngine) tokenValue() string {
	return ce.tknzr.token().value
}

func (ce *compilationEngine) check(expected string) {
	switch expected {
	case "varName", "className", "subroutineName", "identifier":
		if ce.tknzr.token().lex != Identifier {
			ce.expected(expected)
		}
	case "constant":
		if ce.tknzr.token().lex != StringConst && ce.tknzr.token().lex != IntConst {
			ce.expected(expected)
		}
	case "constructor", "function", "method":
		if ce.tknzr.token().lex != Keyword {
			ce.expected(expected)
		}
	case "type":
		if ce.tknzr.token().lex == Keyword {
			if ce.tokenValue() != "int" &&
				ce.tokenValue() != "boolean" && ce.tokenValue() != "char" {
				ce.expected("type : int, boolean or char")
			}
		} else if ce.tknzr.token().lex != Identifier {
			// className
			ce.expected("type or className")
		}
	default:
		if expected != ce.tokenValue() {
			ce.expected(expected)
		}
	}

	// advance tokenizer
	ce.tknzr.advance()
}

func (ce *compilationEngine) debug() {
	if !ce.isDebugEnabled {
		return
	}
}
