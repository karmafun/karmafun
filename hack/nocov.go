// cSpell: words Stmts fset forbidigo gosec gocyclo wrapcheck
//
//nolint:forbidigo,gosec,gocyclo,wrapcheck,exhaustive // Vibe coded tool
package main

import (
	"bufio"
	"errors"
	"fmt"
	"go/parser"
	"go/scanner"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type lineRange struct {
	start int
	end   int
}

type fileAnalysis struct {
	nocovLines map[int]struct{}
	blocks     []lineRange
}

type coverageEntry struct {
	filePath   string
	original   string
	startLine  int
	startCol   int
	endLine    int
	endCol     int
	numStmts   int
	execCount  int
	lineNumber int
}

var coverageLineRE = regexp.MustCompile(`^(.+):(\d+)\.(\d+),(\d+)\.(\d+)\s+(\d+)\s+(\d+)$`)

func main() {
	if len(os.Args) < 2 || len(os.Args) > 3 {
		fmt.Fprintf(os.Stderr, "usage: %s <coverage-file> [output-file]\n", os.Args[0])
		os.Exit(2)
	}

	inputPath := os.Args[1]
	outputPath := inputPath + ".nocov"
	if len(os.Args) == 3 {
		outputPath = os.Args[2]
	}

	moduleName, err := readModuleName("go.mod")
	if err != nil {
		die("read module name from go.mod", err)
	}

	lines, err := readLines(inputPath)
	if err != nil {
		die("read coverage profile", err)
	}
	if len(lines) == 0 {
		die("read coverage profile", errors.New("empty coverage file"))
	}

	mode := lines[0]
	if !strings.HasPrefix(mode, "mode:") {
		die("parse coverage profile", fmt.Errorf("first line must start with mode:, got %q", mode))
	}

	analyses := make(map[string]*fileAnalysis)
	ignoredByFile := make(map[string]int)
	totalIgnored := 0

	outLines := make([]string, 0, len(lines))
	outLines = append(outLines, mode)

	for i := 1; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}

		entry, err := parseCoverageEntry(lines[i], i+1)
		if err != nil {
			die("parse coverage profile", err)
		}

		if entry.execCount != 0 {
			outLines = append(outLines, lines[i])
			continue
		}

		sourcePath, relPath, err := resolveSourcePath(moduleName, entry.filePath)
		if err != nil {
			die("resolve source path", fmt.Errorf("line %d: %w", entry.lineNumber, err))
		}

		analysis, found := analyses[relPath]
		if !found {
			analysis, err = buildAnalysis(sourcePath)
			if err != nil {
				die("analyze source file", fmt.Errorf("%s: %w", relPath, err))
			}
			analyses[relPath] = analysis
		}

		if shouldIgnore(entry, analysis) {
			ignoredByFile[relPath]++
			totalIgnored++
			continue
		}

		outLines = append(outLines, lines[i])
	}

	if err := writeLines(outputPath, outLines); err != nil {
		die("write filtered coverage profile", err)
	}

	fmt.Printf("wrote filtered coverage profile: %s\n", outputPath)
	printSummary(ignoredByFile, totalIgnored)
}

func readModuleName(goModPath string) (string, error) {
	lines, err := readLines(goModPath)
	if err != nil {
		return "", err
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "module ") {
			module := strings.TrimSpace(strings.TrimPrefix(trimmed, "module "))
			if module == "" {
				return "", errors.New("empty module declaration")
			}
			return module, nil
		}
	}

	return "", errors.New("module declaration not found")
}

func parseCoverageEntry(line string, lineNumber int) (*coverageEntry, error) {
	matches := coverageLineRE.FindStringSubmatch(strings.TrimSpace(line))
	if matches == nil {
		return nil, fmt.Errorf("line %d: invalid coverage entry %q", lineNumber, line)
	}

	toInt := func(idx int) (int, error) {
		v, err := strconv.Atoi(matches[idx])
		if err != nil {
			return 0, fmt.Errorf("line %d: parse integer %q: %w", lineNumber, matches[idx], err)
		}
		return v, nil
	}

	startLine, err := toInt(2)
	if err != nil {
		return nil, err
	}
	startCol, err := toInt(3)
	if err != nil {
		return nil, err
	}
	endLine, err := toInt(4)
	if err != nil {
		return nil, err
	}
	endCol, err := toInt(5)
	if err != nil {
		return nil, err
	}
	numStmts, err := toInt(6)
	if err != nil {
		return nil, err
	}
	execCount, err := toInt(7)
	if err != nil {
		return nil, err
	}

	return &coverageEntry{
		filePath:   matches[1],
		startLine:  startLine,
		startCol:   startCol,
		endLine:    endLine,
		endCol:     endCol,
		numStmts:   numStmts,
		execCount:  execCount,
		original:   line,
		lineNumber: lineNumber,
	}, nil
}

func resolveSourcePath(moduleName, coveredPath string) (string, string, error) {
	relPath := coveredPath
	prefix := moduleName + "/"
	if strings.HasPrefix(coveredPath, prefix) {
		relPath = strings.TrimPrefix(coveredPath, prefix)
	}

	relPath = filepath.Clean(relPath)

	if _, err := os.Stat(relPath); err == nil {
		return relPath, relPath, nil
	}

	return "", "", fmt.Errorf("cannot locate source file for %q (resolved as %q)", coveredPath, relPath)
}

func buildAnalysis(sourcePath string) (*fileAnalysis, error) {
	content, err := os.ReadFile(sourcePath)
	if err != nil {
		return nil, err
	}

	source := string(content)
	lines := strings.Split(source, "\n")

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, sourcePath, source, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	nocovLines := make(map[int]struct{})
	for _, group := range file.Comments {
		for _, c := range group.List {
			if !strings.HasPrefix(c.Text, "//") {
				continue
			}
			commentText := strings.TrimSpace(strings.TrimPrefix(c.Text, "//"))
			if !strings.HasPrefix(commentText, "nocov") {
				continue
			}
			line := fset.Position(c.Slash).Line
			nocovLines[line] = struct{}{}
		}
	}

	braceMap := buildBraceMap(sourcePath, source)

	blocks := make([]lineRange, 0)
	seenBlocks := make(map[lineRange]struct{})
	for nocovLine := range nocovLines {
		for _, candidate := range []int{nocovLine, nocovLine + 1} {
			if candidate <= 0 || candidate > len(lines) {
				continue
			}
			if !lineStartsBlock(lines[candidate-1]) {
				continue
			}
			endLine, ok := pickBlockEnd(braceMap, candidate)
			if !ok {
				continue
			}
			r := lineRange{start: candidate, end: endLine}
			if _, exists := seenBlocks[r]; exists {
				continue
			}
			seenBlocks[r] = struct{}{}
			blocks = append(blocks, r)
		}
	}

	sort.Slice(blocks, func(i, j int) bool {
		if blocks[i].start == blocks[j].start {
			return blocks[i].end < blocks[j].end
		}
		return blocks[i].start < blocks[j].start
	})

	return &fileAnalysis{
		nocovLines: nocovLines,
		blocks:     blocks,
	}, nil
}

func buildBraceMap(filename, source string) map[int][]int {
	fset := token.NewFileSet()
	file := fset.AddFile(filename, -1, len(source))

	var s scanner.Scanner
	s.Init(file, []byte(source), nil, 0)

	type openBrace struct {
		line int
	}

	stack := make([]openBrace, 0)
	braceMap := make(map[int][]int)

	for {
		pos, tok, _ := s.Scan()
		if tok == token.EOF {
			break
		}

		line := file.Position(pos).Line
		switch tok {
		case token.LBRACE:
			stack = append(stack, openBrace{line: line})
		case token.RBRACE:
			if len(stack) == 0 {
				continue
			}
			open := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			braceMap[open.line] = append(braceMap[open.line], line)
		}
	}

	return braceMap
}

func lineStartsBlock(line string) bool {
	trimmed := strings.TrimSpace(line)
	trimmed = strings.TrimSuffix(trimmed, "//nocov")
	trimmed = strings.TrimSpace(trimmed)
	return strings.HasSuffix(trimmed, "{")
}

func pickBlockEnd(braceMap map[int][]int, startLine int) (int, bool) {
	ends := braceMap[startLine]
	if len(ends) == 0 {
		return 0, false
	}

	end := ends[0]
	for _, v := range ends[1:] {
		if v > end {
			end = v
		}
	}
	return end, true
}

func shouldIgnore(entry *coverageEntry, analysis *fileAnalysis) bool {
	if _, ok := analysis.nocovLines[entry.startLine]; ok {
		return true
	}
	if entry.startLine > 1 {
		if _, ok := analysis.nocovLines[entry.startLine-1]; ok {
			return true
		}
	}

	for _, block := range analysis.blocks {
		if rangesIntersect(entry.startLine, entry.endLine, block.start, block.end) {
			return true
		}
	}

	return false
}

func rangesIntersect(aStart, aEnd, bStart, bEnd int) bool {
	return aStart <= bEnd && bStart <= aEnd
}

func readLines(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	//nolint:errcheck // Intentionally ignore error on close since we're exiting immediately after
	defer f.Close()

	lines := make([]string, 0)
	s := bufio.NewScanner(f)
	for s.Scan() {
		lines = append(lines, s.Text())
	}
	if err := s.Err(); err != nil {
		return nil, err
	}
	return lines, nil
}

func writeLines(path string, lines []string) error {
	content := strings.Join(lines, "\n") + "\n"
	return os.WriteFile(path, []byte(content), 0o644)
}

func printSummary(ignoredByFile map[string]int, totalIgnored int) {
	if len(ignoredByFile) == 0 {
		fmt.Println("ignored lines: 0")
		return
	}

	keys := make([]string, 0, len(ignoredByFile))
	for k := range ignoredByFile {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	fmt.Println("ignored lines by file:")
	for _, k := range keys {
		fmt.Printf("  %s: %d\n", k, ignoredByFile[k])
	}
	fmt.Printf("total ignored lines: %d\n", totalIgnored)
}

func die(action string, err error) {
	fmt.Fprintf(os.Stderr, "error: %s: %v\n", action, err)
	os.Exit(1)
}
