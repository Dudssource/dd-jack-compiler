package compiler

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strings"
)

type JackAnalyser struct {
	tknzr   *jackTokenizer
	dstFile io.Writer
}

func NewJackAnalyser(srcFile io.Reader, dstFile io.Writer) *JackAnalyser {
	return &JackAnalyser{
		tknzr:   newTokenizer(srcFile),
		dstFile: dstFile,
	}
}

func (anlzr *JackAnalyser) Run() error {
	if strings.EqualFold(os.Getenv("JACK_DUMP_TOKENS"), "true") {
		fmt.Fprint(anlzr.dstFile, "<tokens>")
		for token, hasNext := anlzr.tknzr.getNextToken(); hasNext; token, hasNext = anlzr.tknzr.getNextToken() {
			fmt.Fprintf(anlzr.dstFile, "<%s>", token.lex)
			_ = xml.EscapeText(anlzr.dstFile, []byte(token.value))
			fmt.Fprintf(anlzr.dstFile, "</%s>\n", token.lex)
		}
		fmt.Fprint(anlzr.dstFile, "</tokens>")
		return nil
	}
	return newCompilationEngine(anlzr.tknzr, anlzr.dstFile).compile()
}
