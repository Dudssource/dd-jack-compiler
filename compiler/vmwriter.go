package compiler

import (
	"fmt"
	"io"
)

type vmWriter struct {
	dstFile io.Writer
}

func (w *vmWriter) writePush(segment string, position int) {
	w.write("\tpush %s %d", segment, position)
}

func (w *vmWriter) writePop(segment string, position int) {
	w.write("\tpop %s %d", segment, position)
}

func (w *vmWriter) writeArithmetic(command string) {
	w.write("\t%s", command)
}

func (w *vmWriter) writeLabel(label string) {
	w.write("label %s", label)
}

func (w *vmWriter) writeUnaryOp(op string) {
	switch op {
	case "-":
		w.write("\tneg")
	case "~":
		w.write("\tnot")
	}
}

func (w *vmWriter) writeOp(op string) {
	switch op {
	case "-":
		w.write("\tsub")
	case "*":
		w.write("\tcall Math.multiply 2")
	case "/":
		w.write("\tcall Math.divide 2")
	case "+":
		w.write("\tadd")
	case "|":
		w.write("\tor")
	case "&":
		w.write("\tand")
	case "<":
		w.write("\tlt")
	case ">":
		w.write("\tgt")
	case "=":
		w.write("\teq")
	}
}

func (w *vmWriter) writeGoto(label string) {
	w.write("\tgoto %s", label)
}

func (w *vmWriter) writeIf(label string) {
	w.write("\tif-goto %s", label)
}

func (w *vmWriter) writeCall(name string, nArgs int) {
	w.write("\tcall %s %d", name, nArgs)
}

func (w *vmWriter) writeFunction(name string, nLocals int) {
	w.write("function %s %d", name, nLocals)
}

func (w *vmWriter) writeReturn() {
	w.write("\treturn")
}

func (w *vmWriter) write(format string, args ...any) {
	_, _ = fmt.Fprintf(w.dstFile, format+"\n", args...)
}
