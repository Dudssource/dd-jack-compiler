# Jack Compiler (Golang)

Simple (zero dependency), fully implemented, working implementation of the Jack Compiler for the HACK VM language, as part of the course [Nand to Tetris](https://www.nand2tetris.org/), written in Golang.

## Grammar
 
![grammar](./docs/grammar.png)


## How to use

The `testdata` folder includes a few examples of valid Jack (.jack) files and VM (.vm) files.

In order to run the compiler, the following is required:

* Golang >= v1.25.0

Example for single Jack files:

```shell
go run main.go testdata/compiler/Seven/A/Main.jack
```

Example for multi Jack file folder:

```shell
go run main.go testdata/compiler/Seven/A/
```

Usage:

```plaintext
Usage of JackCompiler:
		JackCompiler myProg/FileName.jack
		JackCompiler myProg/
```

For every jack file, the program will generate a VM file on the same path `vm\testdata\FileName.vm`.

## Screenshot

![hackasm-example](./docs/screenshot.png)

## Building

In order to submit projects to Coursera, I decided to create a fake Makefile and submit the Linux executable directly. In order to make that portable (I'm running Windows :confounded:), I created a Dockerfile to generate the proper binary (it's better than trying go multiplatform build).

Just run:

```shell
# building
docker build -t dd-jack-compiler .
# running
docker run -d dd-jack-compiler
# grab the container ID and run
docker cp $CONTAINER:/app/JackCompiler .
```

## Copyright

The Jack/HACK VM Language and it's specification, are part of the course Nand to Tetris ([https://www.nand2tetris.org/](https://www.nand2tetris.org/)) - copyright Noam Nisan and Shimon Schocken