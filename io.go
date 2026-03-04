package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"os"
)

// file models the subset of *os.File used by the CLI.
type file interface {
	Read(b []byte) (int, error)
	Write(b []byte) (int, error)
	Seek(offset int64, whence int) (int64, error)
	Truncate(size int64) error
	Close() error
}

// osOpenFile matches the signature of os.OpenFile.
type osOpenFile func(name string, flag int, perm os.FileMode) (file, error)

// openFile wraps os.OpenFile for testing.
var openFile osOpenFile = func(
	name string, flag int, perm os.FileMode,
) (file, error) {
	return os.OpenFile(name, flag, perm)
}

// config holds parsed CLI arguments.
type config struct {
	version bool
	write   bool
	paths   []string
}

const usageText = "usage: gouse [-v] [-w] [file paths...]"

// parseArgs parses CLI arguments and returns the config, the usage text, and
// any parse error. It returns flag.ErrHelp for -h, -help, --help, and related
// help-triggering misuse.
func parseArgs(args []string) (*config, string, error) {
	c := new(config)
	flags := flag.NewFlagSet("", flag.ContinueOnError)
	var out bytes.Buffer
	flags.SetOutput(&out)
	flags.BoolVar(&c.version, "v", false, "show version")
	flags.BoolVar(&c.write, "w", false, "write results to files")
	flags.Usage = func() { out.Write([]byte(usageText)) }
	if err := flags.Parse(args); err != nil {
		return nil, out.String(), err
	}
	// Call flags.Args only after flags.Parse.
	c.paths = flags.Args()
	return c, out.String(), nil
}

// toggleFile reads from in, toggles the code, replaces out when it is the same
// file, and writes the toggled bytes.
func toggleFile(ctx context.Context, in, out file) error {
	code, err := io.ReadAll(in)
	if err != nil {
		return fmt.Errorf("toggleFile: in io.ReadAll: %w", err)
	}
	toggled, err := toggle(ctx, code)
	if err != nil {
		return fmt.Errorf("toggleFile: %w", err)
	}
	if out == in {
		if _, err := out.Seek(0, io.SeekStart); err != nil {
			return fmt.Errorf("toggleFile: in file.Seek: %w", err)
		}
		if err := out.Truncate(0); err != nil {
			return fmt.Errorf(
				"toggleFile: in file.Truncate: %w", err,
			)
		}
	}
	if _, err := out.Write(toggled); err != nil {
		return fmt.Errorf("toggleFile: in file.Write: %w", err)
	}
	return nil
}
