// This file is part of DD Jack Compiler.
// Copyright (C) 2025-2025 Eduardo <dudssource@gmail.com>
//
// Jack Compiler is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Jack Compiler is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Jack Compiler.  If not, see <http://www.gnu.org/licenses/>.
package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/Dudssource/dd-jack-compiler/compiler"
)

func main() {

	// parse args
	args := os.Args

	// validate src
	if len(args) != 2 {
		log.Println(`Usage of jackcompiler:
		JackCompiler myProg/FileName.jack
		JackCompiler myProg/`)
		os.Exit(0)
	}

	var (

		// src path
		srcPath = args[1]
		// error list
		errorList = make([]error, 0)
	)

	// clean up src path
	srcPath = strings.TrimRight(srcPath, string(os.PathSeparator))

	// stat
	srcInfo, err := os.Stat(srcPath)
	if err != nil {
		log.Fatalf("stat %s : %s", srcPath, err.Error())
	}

	// src is a directory, transverse to get all *.jack files
	if srcInfo.IsDir() {

		// all matching vm files within directory
		matches, err := filepath.Glob(fmt.Sprintf("%s/*.jack", strings.TrimRight(srcPath, string(os.PathSeparator))))
		if err != nil {
			log.Fatalf("glob %s", err.Error())
		}

		// translate all files
		for _, srcPath := range matches {
			// file name (without extension)
			dstPath := strings.Split(filepath.Base(srcPath), ".")[0]
			// analyse file
			if err := analyse(srcPath, dstPath); err != nil {
				errorList = append(errorList, fmt.Errorf("%s : %w", srcPath, err))
			}
		}

	} else {

		// file name (without extension)
		dstPath := strings.Split(filepath.Base(srcPath), ".")[0]

		// analyse file
		if err := analyse(srcPath, dstPath); err != nil {
			errorList = append(errorList, fmt.Errorf("%s : %w", srcPath, err))
		}
	}

	if len(errorList) > 0 {
		log.Fatalf("errors found : %s", errors.Join(errorList...).Error())
	}
}

func analyse(srcPath, dstPath string) error {

	// open src file
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	// create dst file
	finalDstPath := filepath.Join(filepath.Dir(srcPath), dstPath+".vm")
	dstFile, err := os.Create(finalDstPath)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	// run the analyser
	if err := compiler.NewJackAnalyser(srcFile, dstFile).Run(); err != nil {
		return err
	}

	// ok
	log.Printf("JACK Compiler finished successfully, output to %s\n", finalDstPath)

	return nil
}
