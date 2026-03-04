package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestToggle(t *testing.T) {
	inputsPaths, err := filepath.Glob(filepath.Join("testdata", "*.input"))
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	t.Cleanup(cancel)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range inputsPaths {
		p := p
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

const getSymbolsInfoFromBuildErrorsInput = `
	package p

	func main() {
	var (
		notUsed0 = false
		used0    bool
	)
	notUsed1, used1 := "", "", "" // more values than variables
	_, _ = used0, used1 // no closing brace
`

func TestGetSymbolsInfoFromBuildErrors(t *testing.T) {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	t.Cleanup(cancel)
	t.Run("ignore other errors", func(t *testing.T) {
		t.Parallel()
		input := []byte(
			getSymbolsInfoFromBuildErrorsInput,
		)
		want := []symbolInfo{
			{"notUsed0", 5},
			{"notUsed1", 8},
		}
		got, err := getSymbolsInfoFromBuildErrors(
			ctx, input, notUsedErrorRegexpSuffix,
		)
		if err != nil {
			t.Fatal(err)
		}
		for i, info := range got {
			if info.name != want[i].name {
				t.Errorf(
					"got: %s, want: %s",
					info.name,
					want[i].name,
				)
			}
			if info.lineNum != want[i].lineNum {
				t.Errorf(
					"got: %d, want: %d",
					info.lineNum,
					want[i].lineNum,
				)
			}
		}
	})
	t.Run("pre-canceled context", func(t *testing.T) {
		t.Parallel()
		canceledCtx, cancel := context.WithCancel(
			context.Background(),
		)
		cancel()
		got, err := getSymbolsInfoFromBuildErrors(
			canceledCtx,
			[]byte("package p\n"),
			notUsedErrorRegexpSuffixWithColon,
		)
		if err != nil {
			t.Fatalf("got: %v, want: nil", err)
		}
		if got != nil {
			t.Fatalf("got: %v, want: nil", got)
		}
	})
}
