package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

const (
	fakeUsageSuffix = " /* TODO: gouse */"
	fakeUsagePrefix = "; _ ="

	// These suffixes match current `go build` diagnostics. They are
	// heuristics, not a stable API.
	noProviderErrorSuffix = "no required module provides package"
	commentPrefix         = "// "
	notUsedErrorSuffix    = "declared and not used"
)

var (
	escapedFakeUsageSuffix = regexp.QuoteMeta(fakeUsageSuffix)
	fakeUsage              = regexp.MustCompile(
		fakeUsagePrefix + ".*" + escapedFakeUsageSuffix,
	)
	fakeUsageAfterGofmt = regexp.MustCompile(
		`\s*_\s*=\s*[_\pL][_\pL\pN]*\s*` + escapedFakeUsageSuffix,
	)
	notUsedErrorWithColonSuffix = notUsedErrorSuffix + ":"
)

// toggle removes existing fake usages or inserts new ones based on build
// diagnostics.
func toggle(ctx context.Context, code []byte) ([]byte, error) {
	// fakeUsage must run before fakeUsageAfterGofmt because it also removes the
	// leading ‘;’.
	if fakeUsage.Match(code) {
		return fakeUsage.ReplaceAll(code, []byte("")), nil
	}
	if fakeUsageAfterGofmt.Match(code) {
		return fakeUsageAfterGofmt.ReplaceAll(code, []byte("")), nil
	}

	lines := bytes.Split(code, []byte("\n"))
	// Comment out imports that fail with a missing module diagnostic and store
	// their line numbers.
	importsWithoutProviderInfo, err := getSymbolsInfoFromBuildErrors(
		ctx, code, noProviderErrorSuffix,
	)
	if err != nil {
		return nil, fmt.Errorf("toggle: %w", err)
	}
	var commentedLinesNums []int
	for _, info := range importsWithoutProviderInfo {
		l := &lines[info.lineNum]
		*l = append([]byte(commentPrefix), *l...)
		commentedLinesNums = append(commentedLinesNums, info.lineNum)
	}
	// Get ‘declared and not used’ diagnostics and insert fake usages for the
	// reported names.
	notUsedVarsInfo, err := getSymbolsInfoFromBuildErrors(
		ctx,
		bytes.Join(lines, []byte("\n")),
		notUsedErrorWithColonSuffix,
	)
	if err != nil {
		return nil, fmt.Errorf("toggle: %w", err)
	}
	for _, info := range notUsedVarsInfo {
		l := &lines[getFakeUsageLineNum(lines, info.lineNum)]
		*l = appendBeforeTrailingCR(*l, []byte(
			fakeUsagePrefix+info.name+fakeUsageSuffix,
		))
	}
	// Restore the commented imports.
	for _, line := range commentedLinesNums {
		l := &lines[line]
		uncommentedLine := []rune(
			string(*l),
		)[len([]rune(commentPrefix)):]
		*l = []byte(string(uncommentedLine))
	}
	return bytes.Join(lines, []byte("\n")), nil
}

func appendBeforeTrailingCR(line, extra []byte) []byte {
	if len(line) == 0 || line[len(line)-1] != '\r' {
		return append(line, extra...)
	}
	line = append(line[:len(line)-1], extra...)
	return append(line, '\r')
}

func getFakeUsageLineNum(lines [][]byte, lineNum int) int {
	if !isSwitchHeaderLine(lines[lineNum]) {
		return lineNum
	}
	switchClauseLine, ok := getSwitchClauseLineNum(lines, lineNum)
	if !ok {
		return lineNum
	}
	return switchClauseLine
}

func isSwitchHeaderLine(line []byte) bool {
	return bytes.Contains(line, []byte("switch")) &&
		bytes.Contains(line, []byte("{"))
}

func getSwitchClauseLineNum(lines [][]byte, switchLineNum int) (int, bool) {
	for i := switchLineNum + 1; i < len(lines); i++ {
		trimmed := bytes.TrimSpace(lines[i])
		if bytes.HasPrefix(trimmed, []byte("case ")) || bytes.HasPrefix(trimmed, []byte("default:")) {
			return i, true
		}
		if bytes.HasPrefix(trimmed, []byte("}")) {
			return 0, false
		}
	}
	return 0, false
}

// symbolInfo describes a symbol reported by a matching build diagnostic.
type symbolInfo struct {
	name    string
	lineNum int
}

const (
	goFileExt     = ".go"
	lineNumIndex  = 1
	matchEndIndex = 1
)

var (
	// symbolPositionInError matches the ‘.go:line:column: ’ segment in a build
	// error.
	//
	// Example:
	//
	//	Given a build error ‘.../main[.go:4:2: ]<text of an error>’,
	//	the matched range is denoted with ‘[]’.
	symbolPositionInError = regexp.MustCompile(
		regexp.QuoteMeta(goFileExt) + `:\d+:\d+: `,
	)
)

// getSymbolsInfoFromBuildErrors builds code and returns the symbols reported by
// diagnostics that match the requested suffix.
func getSymbolsInfoFromBuildErrors(
	ctx context.Context, code []byte, suffix string,
) ([]symbolInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	td, err := os.MkdirTemp(os.TempDir(), "gouse")
	if err != nil {
		return nil, fmt.Errorf("getSymbolsInfoFromBuildErrors in os.MkdirTemp: %w", err)
	}
	defer os.RemoveAll(td)

	tf, err := os.CreateTemp(td, "*"+goFileExt)
	if err != nil {
		return nil, fmt.Errorf("getSymbolsInfoFromBuildErrors: in os.CreateTemp: %w", err)
	}
	defer tf.Close()

	if _, err := tf.Write(code); err != nil {
		return nil, fmt.Errorf("getSymbolsInfoFromBuildErrors: in temp file write: %w", err)
	}

	goBin, err := exec.LookPath("go")
	if err != nil {
		return nil, fmt.Errorf("getSymbolsInfoFromBuildErrors: in exec.LookPath: %w", err)
	}
	boutput, err := exec.CommandContext(
		ctx,
		goBin, "build", "-o", os.DevNull, tf.Name(),
	).CombinedOutput()
	if err != nil && ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if err == nil {
		return nil, nil
	}

	berrors := strings.Split(string(boutput), "\n")
	var info []symbolInfo
	for _, e := range berrors {
		name, ok := getSymbolNameFromBuildError(e, suffix)
		if !ok {
			continue
		}
		lineNum, err := strconv.Atoi(strings.Split(
			symbolPositionInError.FindString(e), ":",
		)[lineNumIndex])
		if err != nil {
			return nil, fmt.Errorf("getSymbolsInfoFromBuildErrors in strconv.Atoi: %w", err)
		}
		info = append(info, symbolInfo{
			name: name,
			// Convert the reported line number to a 0-based index.
			lineNum: lineNum - 1,
		})
	}
	return info, nil
}

func getSymbolNameFromBuildError(e, suffix string) (string, bool) {
	match := symbolPositionInError.FindStringIndex(e)
	if match == nil {
		return "", false
	}
	afterMatch := e[match[matchEndIndex]:]

	if name, ok := strings.CutPrefix(afterMatch, suffix); ok {
		return name, true
	}
	if suffix != notUsedErrorWithColonSuffix {
		return "", false
	}

	name, ok := strings.CutSuffix(
		afterMatch,
		notUsedErrorSuffix,
	)
	if !ok {
		return "", false
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return "", false
	}
	return " " + name, true
}
