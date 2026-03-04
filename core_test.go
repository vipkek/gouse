package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestToggle(t *testing.T) {
	t.Parallel()

	inputsPaths, err := filepath.Glob(filepath.Join("testdata", "*.input"))
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	t.Cleanup(cancel)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range inputsPaths {
		_, filename := filepath.Split(p)
		testName := filename[:len(filename)-len(filepath.Ext(p))]
		t.Run(testName, func(t *testing.T) {
			t.Parallel()
			input, err := os.ReadFile(p)
			if err != nil {
				t.Fatal(err)
			}
			got, err := toggle(ctx, input)
			if err != nil {
				t.Fatal(err)
			}
			want, err := os.ReadFile(
				filepath.Join("testdata", testName+".golden"),
			)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(got, want) {
				t.Errorf(filesCmpErr, got, want)
			}
		})
	}
}

func TestToggleCRLFInsertion(t *testing.T) {
	t.Parallel()

	t.Run("simple insertion", func(t *testing.T) {
		t.Parallel()
		input := []byte(
			"package p\r\n\r\nfunc main() {\r\n\tx := 1\r\n}\r\n",
		)
		want := []byte(
			"package p\r\n\r\nfunc main() {\r\n" +
				"\tx := 1; _ = x /* TODO: gouse */\r\n}\r\n",
		)

		got, err := toggle(context.Background(), input)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, want) {
			t.Errorf(filesCmpErr, got, want)
		}
		if bytes.Contains(got, []byte("\r; _ =")) {
			t.Fatalf("got malformed CRLF insertion: %q", got)
		}
	})

	t.Run("switch clause insertion", func(t *testing.T) {
		t.Parallel()
		input := []byte(
			"package p\r\n\r\nfunc main() {\r\n" +
				"\tswitch x := 1; {\r\n\tdefault:\r\n\t}\r\n}\r\n",
		)
		want := []byte(
			"package p\r\n\r\nfunc main() {\r\n" +
				"\tswitch x := 1; {\r\n" +
				"\tdefault:; _ = x /* TODO: gouse */\r\n\t}\r\n}\r\n",
		)

		got, err := toggle(context.Background(), input)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, want) {
			t.Errorf(filesCmpErr, got, want)
		}
		if bytes.Contains(got, []byte("\r; _ =")) {
			t.Fatalf("got malformed CRLF insertion: %q", got)
		}
	})
}

const getSymbolsInfoFromBuildErrorsInput = `
	package p

	func main() {
		var notUsed0 missingType
		var notUsed1 int
	}
`

func TestGetSymbolsInfoFromBuildErrors(t *testing.T) {
	t.Parallel()

	t.Run("collects matching diagnostics while ignoring others", func(t *testing.T) {
		t.Parallel()
		want := []symbolInfo{
			{" notUsed0", 4},
			{" notUsed1", 5},
		}
		got, err := getSymbolsInfoFromBuildErrors(
			context.Background(),
			[]byte(getSymbolsInfoFromBuildErrorsInput),
			notUsedErrorWithColonSuffix,
		)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != len(want) {
			t.Fatalf("got: %v, want: %v", got, want)
		}
		if !slices.Equal(got, want) {
			t.Fatalf("got: %v, want: %v", got, want)
		}
	})
	t.Run("pre-canceled context returns context.Canceled", func(t *testing.T) {
		t.Parallel()
		canceledCtx, cancel := context.WithCancel(
			context.Background(),
		)
		cancel()
		got, err := getSymbolsInfoFromBuildErrors(
			canceledCtx,
			[]byte("package p\n"),
			notUsedErrorWithColonSuffix,
		)
		if got != nil {
			t.Fatalf("got: %v, want: nil", got)
		}
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("got: %v, want: %v", err, context.Canceled)
		}
	})
}
