// Command measure_project reports Go and optional frontend source size.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type fileStats struct {
	Lines    int
	Nonblank int
}

type report struct {
	ProductionLines           int
	ProductionNonblankLines   int
	TestLines                 int
	TestNonblankLines         int
	ProductionFiles           int
	TestFiles                 int
	Packages                  int
	PackageNames              []string
	FrontendPresent           bool
	FrontendRoots             []string
	FrontendLines             int
	FrontendNonblankLines     int
	FrontendFiles             int
	FrontendTestLines         int
	FrontendTestNonblankLines int
	FrontendTestFiles         int
}

var ignoredDirs = map[string]bool{
	".git": true, "vendor": true, "node_modules": true, "scripts": true,
	"dist": true, "build": true, "tmp": true, "coverage": true,
}

var frontendExtensions = map[string]bool{
	".css": true, ".js": true, ".jsx": true, ".mjs": true,
	".scss": true, ".svelte": true, ".ts": true, ".tsx": true,
	".vue": true,
}

func main() {
	root := flag.String("root", ".", "repository root")
	enforce := flag.Bool("enforce", false, "fail when hard thresholds are not met")
	frontendRoots := flag.String("frontend-roots", "", "comma-separated frontend roots; auto-detect when empty")
	flag.Parse()

	rootPath, err := filepath.Abs(*root)
	if err != nil {
		fail(err)
	}
	result, err := measure(rootPath)
	if err != nil {
		fail(err)
	}
	if err := measureFrontends(rootPath, *frontendRoots, &result); err != nil {
		fail(err)
	}
	sort.Strings(result.PackageNames)
	payload := map[string]any{
		"production_lines":             result.ProductionLines,
		"production_nonblank_lines":    result.ProductionNonblankLines,
		"test_lines":                   result.TestLines,
		"test_nonblank_lines":          result.TestNonblankLines,
		"production_files":             result.ProductionFiles,
		"test_files":                   result.TestFiles,
		"packages":                     result.Packages,
		"package_names":                result.PackageNames,
		"frontend_present":             result.FrontendPresent,
		"frontend_roots":               result.FrontendRoots,
		"frontend_lines":               result.FrontendLines,
		"frontend_nonblank_lines":      result.FrontendNonblankLines,
		"frontend_files":               result.FrontendFiles,
		"frontend_test_lines":          result.FrontendTestLines,
		"frontend_test_nonblank_lines": result.FrontendTestNonblankLines,
		"frontend_test_files":          result.FrontendTestFiles,
	}
	encoded, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		fail(err)
	}
	fmt.Println(string(encoded))

	if *enforce {
		if result.ProductionLines < 5000 || result.ProductionFiles < 30 || result.Packages < 10 || result.TestLines < 1500 {
			os.Exit(2)
		}
	}
}

func measureFrontends(root, configured string, result *report) error {
	roots, err := resolveFrontendRoots(root, configured)
	if err != nil {
		return err
	}
	result.FrontendPresent = len(roots) > 0
	for _, frontendRoot := range roots {
		relRoot, err := filepath.Rel(root, frontendRoot)
		if err != nil {
			return err
		}
		result.FrontendRoots = append(result.FrontendRoots, filepath.ToSlash(relRoot))
		err = filepath.WalkDir(frontendRoot, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() {
				if path != frontendRoot && ignoredDirs[entry.Name()] {
					return filepath.SkipDir
				}
				return nil
			}
			if !frontendExtensions[strings.ToLower(filepath.Ext(path))] || isGenerated(path) {
				return nil
			}
			stats, err := countFile(path)
			if err != nil {
				return err
			}
			if isFrontendTest(path, frontendRoot) {
				result.FrontendTestFiles++
				result.FrontendTestLines += stats.Lines
				result.FrontendTestNonblankLines += stats.Nonblank
				return nil
			}
			result.FrontendFiles++
			result.FrontendLines += stats.Lines
			result.FrontendNonblankLines += stats.Nonblank
			return nil
		})
		if err != nil {
			return err
		}
	}
	sort.Strings(result.FrontendRoots)
	return nil
}

func resolveFrontendRoots(root, configured string) ([]string, error) {
	var names []string
	if strings.TrimSpace(configured) != "" {
		for _, value := range strings.Split(configured, ",") {
			if name := strings.TrimSpace(value); name != "" {
				names = append(names, name)
			}
		}
	} else {
		names = []string{"frontend", "web", "ui", "client"}
	}

	var roots []string
	for _, name := range names {
		candidate := name
		if !filepath.IsAbs(candidate) {
			candidate = filepath.Join(root, candidate)
		}
		info, err := os.Stat(candidate)
		if err != nil {
			if os.IsNotExist(err) && strings.TrimSpace(configured) == "" {
				continue
			}
			return nil, err
		}
		if !info.IsDir() {
			return nil, fmt.Errorf("frontend root %q is not a directory", candidate)
		}
		absolute, err := filepath.Abs(candidate)
		if err != nil {
			return nil, err
		}
		relative, err := filepath.Rel(root, absolute)
		if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			return nil, fmt.Errorf("frontend root %q is outside repository root", candidate)
		}
		roots = append(roots, absolute)
	}
	return roots, nil
}

func isFrontendTest(path, root string) bool {
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	normalized := "/" + strings.ToLower(filepath.ToSlash(relative))
	base := strings.ToLower(filepath.Base(path))
	return strings.Contains(base, ".test.") || strings.Contains(base, ".spec.") ||
		strings.Contains(normalized, "/__tests__/") || strings.Contains(normalized, "/tests/") ||
		strings.Contains(normalized, "/test/")
}

func measure(root string) (report, error) {
	var result report
	packages := map[string]bool{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			if filepath.ToSlash(rel) == "testdata/generated" {
				return filepath.SkipDir
			}
			if path != root && ignoredDirs[entry.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".go" || isGenerated(path) {
			return nil
		}

		stats, err := countFile(path)
		if err != nil {
			return err
		}
		if strings.HasSuffix(path, "_test.go") {
			result.TestFiles++
			result.TestLines += stats.Lines
			result.TestNonblankLines += stats.Nonblank
			return nil
		}

		result.ProductionFiles++
		result.ProductionLines += stats.Lines
		result.ProductionNonblankLines += stats.Nonblank
		name, err := packageName(path)
		if err != nil {
			return err
		}
		relDir, err := filepath.Rel(root, filepath.Dir(path))
		if err != nil {
			return err
		}
		packages[filepath.ToSlash(filepath.Join(relDir, name))] = true
		return nil
	})
	for name := range packages {
		result.PackageNames = append(result.PackageNames, name)
	}
	result.Packages = len(packages)
	return result, err
}

func countFile(path string) (fileStats, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return fileStats{}, err
	}
	stats := fileStats{}
	for _, b := range data {
		if b == '\n' {
			stats.Lines++
		}
	}
	if len(data) > 0 && data[len(data)-1] != '\n' {
		stats.Lines++
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.TrimSpace(line) != "" {
			stats.Nonblank++
		}
	}
	return stats, nil
}

func packageName(path string) (string, error) {
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.PackageClauseOnly)
	if err != nil {
		return "", err
	}
	return file.Name.Name, nil
}

func isGenerated(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	lines := strings.Split(string(data), "\n")
	if len(lines) > 20 {
		lines = lines[:20]
	}
	header := strings.Join(lines, "\n")
	return strings.Contains(header, "Code generated") && strings.Contains(header, "DO NOT EDIT")
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
