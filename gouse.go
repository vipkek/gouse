// gouse toggles ‘declared and not used’ errors by using idiomatic
// _ = notUsedVar and leaving a TODO comment.
//
// Usage:
//
//	gouse [-v] [-w] [file paths...]
//
// By default, gouse accepts code from stdin or from a file provided as a path
// argument and writes the toggled version to stdout. ‘-w’ flag writes the
// result back to the file. If multiple paths provided, ‘-w’ flag is required.
//
// First it tries to remove previously created fake usages. If there is nothing
// to remove, it tries to build an input and checks the build stdout for
// ‘declared and not used’ errors. If there is any, it creates fake usages for
// unused variables from the errors.
//
// Examples
//
//	$ gouse
//	...input...
//	notUsed = true
//	...input...
//
//	...output...
//	notUsed = true; _ = notUsed /* TODO: gouse */
//	...output...
//
//	$ gouse main.go
//	...
//	notUsed = true; _ = notUsed /* TODO: gouse */
//	...
//
//	$ gouse -w main.go io.go core.go
//	$ cat main.go io.go core.go
//	...
//	notUsedFromMain = true; _ = notUsedFromMain /* TODO: gouse */
//	...
//	notUsedFromIo = true; _ = notUsedFromIo /* TODO: gouse */
//	...
//	notUsedFromCore = true; _ = notUsedFromCore /* TODO: gouse */
//	...
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
)

const (
	errorLogPrefix = "error: "
	logFlag        = 0
	currentVersion = "1.5.3"
)

var (
	errCannotWriteToStdin = errors.New(
		"cannot use ‘-w’ flag with standard input",
	)
	errMustWriteToFiles = errors.New(
		"must use ‘-w’ flag with more than one path",
	)
)

func main() {
	ctx := context.Background()
	os.Exit(run(
		ctx,
		os.Args[1:],
		os.Stdin, os.Stdout, os.Stderr,

		openFile,
	))
}

// run manages logging, parses arguments and toggles the passed files.
func run(
	ctx context.Context,
	args []string,
	stdin, stdout, stderr file,

	openFile osOpenFile,
) int {
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt)
	defer cancel()

	errorLog := log.New(stderr, errorLogPrefix, logFlag)
	infoLog := log.New(stderr, "", logFlag)

	conf, msg, err := parseArgs(args)
	if err == flag.ErrHelp {
		infoLog.Print(msg)
		return 2
	} else if err != nil {
		errorLog.Print(
			fmt.Errorf("run: in parseArgs: %s\n%s", err, msg),
		)
		return 1
	}

	if conf.version {
		infoLog.Print(currentVersion)
		return 0
	}

	if len(conf.paths) == 0 {
		if conf.write {
			errorLog.Print(errCannotWriteToStdin)
			return 1
		}
		if err := toggleFile(ctx, stdin, stdout); err != nil {
			errorLog.Print(err)
			return 1
		}
		return 0
	}
	if len(conf.paths) > 1 && !conf.write {
		errorLog.Print(errMustWriteToFiles)
		return 1
	}
	for _, p := range conf.paths {
		if err := processPath(
			ctx,
			p,
			conf.write,
			stdout,
			openFile,
		); err != nil {
			errorLog.Print(err)
			return 1
		}
	}
	return 0
}

func processPath(
	ctx context.Context,
	path string,
	write bool,
	stdout file,
	openFile osOpenFile,
) error {
	var in file
	var out *file
	var access int
	if write {
		out = &in
		access = os.O_RDWR
	} else {
		out = &stdout
		access = os.O_RDONLY
	}
	var err error
	in, err = openFile(path, access, os.ModeExclusive)
	if err != nil {
		return err
	}
	defer in.Close()
	return toggleFile(ctx, in, *out)
}
