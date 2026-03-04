package main

import (
	"bytes"
	"errors"
	"flag"
	"strings"
	"testing"
)

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
			// That’s the test where all is nil.
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
		t.Run(strings.Join(test.args, " "), func(t *testing.T) {
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
			if (*conf).version != wantConf.version {
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
			bpaths := []byte(strings.Join(conf.paths, ""))
			wantConfBPaths := []byte(
				strings.Join(wantConf.paths, ""),
			)
			if !bytes.Equal(bpaths, wantConfBPaths) {
				t.Errorf(
					"got: %v, want: %v",
					bpaths,
					wantConfBPaths)
			}
		})
	}
}
