package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
)

func makeCleanupBenchmarkInput(raw []byte) []byte {
	lines := bytes.Split(raw, []byte("\n"))
	marker := []byte(fakeUsagePrefix + " _gouseBench" + fakeUsageSuffix)
	for i, line := range lines {
		withMarker := append([]byte(nil), line...)
		withMarker = append(withMarker, marker...)
		lines[i] = withMarker
	}
	return bytes.Join(lines, []byte("\n"))
}

func BenchmarkToggleLargeFixtureCleanup(b *testing.B) {
	raw, err := os.ReadFile(filepath.Join("testdata", "bench", "input"))
	if err != nil {
		b.Fatal(err)
	}
	input := makeCleanupBenchmarkInput(raw)
	if len(input) <= len(raw) {
		b.Fatalf("input length got: %d, want > %d", len(input), len(raw))
	}

	ctx := context.Background()
	b.ReportAllocs()
	b.SetBytes(int64(len(input)))
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			got, err := toggle(ctx, input)
			if err != nil {
				b.Fatal(err)
			}
			if len(got) >= len(input) {
				b.Fatalf("output length got: %d, want < %d", len(got), len(input))
			}
		}
	})
}

func BenchmarkGetSymbolNameFromBuildError(b *testing.B) {
	benchmarks := []struct {
		name     string
		errText  string
		suffix   string
		wantName string
	}{
		{
			name:     "with_colon_suffix",
			errText:  "/tmp/bench.go:4:2: " + notUsedErrorWithColonSuffix + " benchVar",
			suffix:   notUsedErrorWithColonSuffix,
			wantName: " benchVar",
		},
		{
			name:     "without_colon_suffix",
			errText:  "/tmp/bench.go:4:2: benchVar " + notUsedErrorSuffix,
			suffix:   notUsedErrorWithColonSuffix,
			wantName: " benchVar",
		},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			b.ReportAllocs()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					got, ok := getSymbolNameFromBuildError(
						bm.errText,
						bm.suffix,
					)
					if !ok {
						b.Fatal("got: false, want: true")
					}
					if got != bm.wantName {
						b.Fatalf("got: %q, want: %q", got, bm.wantName)
					}
				}
			})
		})
	}
}
