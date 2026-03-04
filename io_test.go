package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestToggleFileRewritesRealFileInPlace(t *testing.T) {
	t.Parallel()

	input, err := os.ReadFile(filepath.Join("testdata", "not_used.input"))
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile(filepath.Join("testdata", "not_used.golden"))
	if err != nil {
		t.Fatal(err)
	}

	f, err := os.CreateTemp(t.TempDir(), "*.go")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if closeErr := f.Close(); closeErr != nil {
			t.Error(closeErr)
		}
	})

	if _, err := f.Write(input); err != nil {
		t.Fatal(err)
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	if err := toggleFile(context.Background(), f, f); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(f.Name())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf(filesCmpErr, got, want)
	}

	if _, err := f.Seek(0, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	if err := toggleFile(context.Background(), f, f); err != nil {
		t.Fatal(err)
	}

	got, err = os.ReadFile(f.Name())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, input) {
		t.Errorf(filesCmpErr, got, input)
	}
}

func TestParseArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		args   []string
		conf   config
		output string
		err    error
	}{
		{
			args: []string{"-v"},
			conf: config{
				version: true,
				write:   false,
				paths:   []string{},
			},
		},
		{
			args:   []string{"-h"},
			output: usageText,
			err:    flag.ErrHelp,
		},
		{
			args:   []string{"-help"},
			output: usageText,
			err:    flag.ErrHelp,
		},
		{
			args:   []string{"--help"},
			output: usageText,
			err:    flag.ErrHelp,
		},
		{
			// That’s the test where everything is nil.
		},
		{
			args: []string{"-w"},
			conf: config{
				version: false,
				write:   true,
				paths:   []string{},
			},
		},
		{
			args: []string{"path1", "path2"},
			conf: config{
				version: false,
				write:   false,
				paths:   []string{"path1", "path2"},
			},
		},
	}
	for _, tt := range tests {
		test := tt
		testName := strings.Join(test.args, " ")
		if testName == "" {
			testName = "empty"
		}
		t.Run(testName, func(t *testing.T) {
			t.Parallel()
			conf, output, err := parseArgs(test.args)

			if !errors.Is(err, test.err) {
				t.Errorf("got: %v, want: %v", err, test.err)
				if test.output == "" {
					return
				}
			}
			if test.output != "" {
				if output != test.output {
					t.Errorf(
						"got: %s, want: %s",
						output,
						test.output,
					)
				}
				return
			}

			wantConf := test.conf
			if conf == nil {
				t.Fatal("got: nil, want: non-nil")
			}
			if conf.version != wantConf.version {
				t.Errorf(
					"got: %t, want: %t",
					conf.version,
					wantConf.version,
				)
			}
			if conf.write != wantConf.write {
				t.Errorf(
					"got: %t, want: %t",
					conf.write,
					wantConf.write,
				)
			}
			if !slices.Equal(conf.paths, wantConf.paths) {
				t.Errorf(
					"got: %v, want: %v",
					conf.paths,
					wantConf.paths,
				)
			}
		})
	}
}
