package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const filesCmpErr = `
========= got:
%s
========= want:
%s`

type fakeFile struct {
	contents   []byte
	offset     int64
	closeCount int
	closed     bool
}

var (
	errFakeFileNegativeOffset    = errors.New("fakeFile.Seek: negative offset")
	errFakeFileUnsupportedWhence = errors.New("fakeFile.Seek: only io.SeekStart is supported")
	errFakeFileNegativeSize      = errors.New("fakeFile.Truncate: negative size")
)

func newFakeFile(buf ...byte) *fakeFile {
	f := fakeFile{contents: append([]byte(nil), buf...)}
	return &f
}

func (f *fakeFile) Read(b []byte) (int, error) {
	start := int(f.offset)
	if start >= len(f.contents) {
		return 0, io.EOF
	}
	n := copy(b, f.contents[start:])
	f.offset += int64(n)
	return n, nil
}

func (f *fakeFile) Write(b []byte) (int, error) {
	start := int(f.offset)
	if start > len(f.contents) {
		f.contents = append(
			f.contents,
			make([]byte, start-len(f.contents))...,
		)
	}
	end := start + len(b)
	if end > len(f.contents) {
		f.contents = append(
			f.contents,
			make([]byte, end-len(f.contents))...,
		)
	}
	copy(f.contents[start:end], b)
	f.offset = int64(end)
	return len(b), nil
}

func (f *fakeFile) Seek(offset int64, whence int) (int64, error) {
	if whence != io.SeekStart {
		return 0, errFakeFileUnsupportedWhence
	}
	if offset < 0 {
		return 0, errFakeFileNegativeOffset
	}
	f.offset = offset
	return f.offset, nil
}

func (f *fakeFile) Truncate(size int64) error {
	if size < 0 {
		return errFakeFileNegativeSize
	}

	newLen := int(size)
	switch {
	case newLen < len(f.contents):
		f.contents = f.contents[:newLen]
	case newLen > len(f.contents):
		f.contents = append(
			f.contents,
			make([]byte, newLen-len(f.contents))...,
		)
	}
	if f.offset > size {
		f.offset = size
	}
	return nil
}

func (f *fakeFile) Close() error {
	if f.closed {
		return nil
	}
	f.closed = true
	f.closeCount++
	return nil
}

func (f *fakeFile) Bytes() []byte {
	return append([]byte(nil), f.contents...)
}

func (f *fakeFile) reopen() error {
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return err
	}
	f.closed = false
	return nil
}

func TestRun(t *testing.T) {
	t.Parallel()

	input, err := os.ReadFile(filepath.Join("testdata", "not_used.input"))
	var openInput osOpenFile = func(
		name string, flag int, perm os.FileMode,
	) (file, error) {
		return newFakeFile(input...), nil
	}
	if err != nil {
		t.Fatal(err)
	}
	mockPath := "filename"
	tests := []struct {
		args         []string
		wantFilename string
		wantOutput   string
		wantStatus   int
	}{
		{
			args:       []string{"-v"},
			wantOutput: currentVersion + "\n",
			wantStatus: 0,
		},
		{
			args:       []string{"-h"},
			wantOutput: usageText + "\n",
			wantStatus: 2,
		},
		{
			args: []string{"-w"},
			wantOutput: errorLogPrefix +
				errCannotWriteToStdin.Error() +
				"\n",
			wantStatus: 1,
		},
		{
			args: []string{mockPath, mockPath},
			wantOutput: errorLogPrefix +
				errMustWriteToFiles.Error() +
				"\n",
			wantStatus: 1,
		},
		{
			args:         []string{},
			wantFilename: "not_used.golden",
			wantStatus:   0,
		},
		{
			args:         []string{mockPath},
			wantFilename: "not_used.golden",
			wantStatus:   0,
		},
	}
	for _, tt := range tests {
		test := tt
		testName := strings.Join(test.args, " ")
		if testName == "" {
			testName = "stdin"
		}
		t.Run(testName, func(t *testing.T) {
			t.Parallel()
			args := test.args
			var (
				stdin  = newFakeFile()
				stdout = newFakeFile()
				stderr = newFakeFile()
			)
			if len(args) == 0 {
				stdin = newFakeFile(input...)
			}
			ctx := context.Background()
			ctx, cancel := context.WithCancel(ctx)
			t.Cleanup(cancel)
			status := run(
				ctx, args, stdin, stdout, stderr, openInput,
			)
			got := stdout.Bytes()
			if test.wantOutput != "" {
				got = append(got, stderr.Bytes()...)
				if !bytes.Equal(got, []byte(test.wantOutput)) {
					t.Errorf(
						filesCmpErr,
						got,
						test.wantOutput,
					)
				}
			} else {
				want, err := os.ReadFile(
					filepath.Join("testdata", test.wantFilename),
				)
				if err != nil {
					t.Fatal(err)
				}
				if !bytes.Equal(got, want) {
					t.Errorf(filesCmpErr, got, want)
				}
			}
			if test.wantStatus != status {
				t.Errorf(
					"got: %d, want: %d",
					status,
					test.wantStatus,
				)
			}
		})
	}
}

func TestRunWriteSinglePathRewritesFile(t *testing.T) {
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
	tempPath := f.Name()
	if _, err := f.Write(input); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	var (
		openCalls int
		gotName   string
		gotFlag   int
		gotPerm   os.FileMode
	)
	openInput := func(
		name string, flag int, perm os.FileMode,
	) (file, error) {
		openCalls++
		gotName = name
		gotFlag = flag
		gotPerm = perm
		return os.OpenFile(name, flag, perm)
	}

	stdout := newFakeFile()
	stderr := newFakeFile()
	status := run(
		context.Background(),
		[]string{"-w", tempPath},
		newFakeFile(),
		stdout,
		stderr,
		openInput,
	)

	if status != 0 {
		t.Fatalf("got: %d, want: 0", status)
	}
	if openCalls != 1 {
		t.Fatalf("open calls got: %d, want: 1", openCalls)
	}
	if gotName != tempPath {
		t.Fatalf("open name got: %q, want: %q", gotName, tempPath)
	}
	if gotFlag != os.O_RDWR {
		t.Fatalf("open flag got: %d, want: %d", gotFlag, os.O_RDWR)
	}
	if gotPerm != os.ModeExclusive {
		t.Fatalf(
			"open perm got: %v, want: %v",
			gotPerm,
			os.ModeExclusive,
		)
	}
	got, err := os.ReadFile(tempPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf(filesCmpErr, got, want)
	}
	if got := stdout.Bytes(); len(got) != 0 {
		t.Fatalf("stdout got: %q, want: empty", got)
	}
	if got := stderr.Bytes(); len(got) != 0 {
		t.Fatalf("stderr got: %q, want: empty", got)
	}
}

func TestRunWriteSamePathTwiceReturnsOriginalContents(t *testing.T) {
	t.Parallel()

	input, err := os.ReadFile(filepath.Join("testdata", "not_used.input"))
	if err != nil {
		t.Fatal(err)
	}

	const mockPath = "filename"
	opened := map[string]*fakeFile{}
	openInput := func(
		name string, flag int, perm os.FileMode,
	) (file, error) {
		if f, ok := opened[name]; ok {
			if err := f.reopen(); err != nil {
				return nil, err
			}
			return f, nil
		}
		f := newFakeFile(input...)
		opened[name] = f
		return f, nil
	}

	status := run(
		context.Background(),
		[]string{"-w", mockPath, mockPath},
		newFakeFile(),
		newFakeFile(),
		newFakeFile(),
		openInput,
	)

	if status != 0 {
		t.Fatalf("got: %d, want: 0", status)
	}

	got, ok := opened[mockPath]
	if !ok {
		t.Fatalf("path %q was not opened", mockPath)
	}
	if !bytes.Equal(got.Bytes(), input) {
		t.Errorf(filesCmpErr, got.Bytes(), input)
	}
}

func TestRunClosesFilesPerIteration(t *testing.T) {
	t.Parallel()

	input, err := os.ReadFile(filepath.Join("testdata", "not_used.input"))
	if err != nil {
		t.Fatal(err)
	}

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		var opened []*fakeFile
		openInput := func(
			name string, flag int, perm os.FileMode,
		) (file, error) {
			if len(opened) > 0 {
				prev := opened[len(opened)-1]
				if prev.closeCount != 1 {
					t.Fatalf(
						"previous file not closed before opening %s",
						name,
					)
				}
			}
			f := newFakeFile(input...)
			opened = append(opened, f)
			return f, nil
		}

		status := run(
			context.Background(),
			[]string{"-w", "a.go", "b.go", "c.go"},
			newFakeFile(),
			newFakeFile(),
			newFakeFile(),
			openInput,
		)

		if status != 0 {
			t.Fatalf("got: %d, want: 0", status)
		}
		if len(opened) != 3 {
			t.Fatalf("got: %d opened files, want: 3", len(opened))
		}
		for i, f := range opened {
			if f.closeCount != 1 {
				t.Errorf(
					"file %d close count got: %d, want: 1",
					i,
					f.closeCount,
				)
			}
		}
	})

	t.Run("later open failure", func(t *testing.T) {
		t.Parallel()

		first := newFakeFile(input...)
		openCalls := 0
		openInput := func(
			name string, flag int, perm os.FileMode,
		) (file, error) {
			openCalls++
			if openCalls == 1 {
				return first, nil
			}
			if first.closeCount != 1 {
				t.Fatalf(
					"first file not closed before failing on %s",
					name,
				)
			}
			return nil, os.ErrNotExist
		}

		status := run(
			context.Background(),
			[]string{"-w", "a.go", "b.go"},
			newFakeFile(),
			newFakeFile(),
			newFakeFile(),
			openInput,
		)

		if status != 1 {
			t.Fatalf("got: %d, want: 1", status)
		}
		if openCalls != 2 {
			t.Fatalf("got: %d open calls, want: 2", openCalls)
		}
		if first.closeCount != 1 {
			t.Fatalf(
				"first file close count got: %d, want: 1",
				first.closeCount,
			)
		}
	})
}
