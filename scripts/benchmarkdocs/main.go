package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type benchmarkDoc struct {
	GOOS       string
	GOARCH     string
	CPU        string
	Benchmarks []benchmarkRow
}

type benchmarkRow struct {
	Name     string
	NsPerOp  string
	BPerOp   string
	AllocsOp string
}

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "usage: %s <output> <benchmark.txt> [benchmark.txt...]\n", filepath.Base(os.Args[0]))
		os.Exit(2)
	}

	outPath := os.Args[1]
	inputs := os.Args[2:]

	docs := make([]benchmarkDoc, 0, len(inputs))
	for _, input := range inputs {
		doc, err := parseBenchmarkFile(input)
		if err != nil {
			fmt.Fprintf(os.Stderr, "parse %s: %v\n", input, err)
			os.Exit(1)
		}
		docs = append(docs, doc)
	}

	sort.Slice(docs, func(i, j int) bool {
		if docs[i].GOOS == docs[j].GOOS {
			return docs[i].GOARCH < docs[j].GOARCH
		}
		return docs[i].GOOS < docs[j].GOOS
	})

	var buf bytes.Buffer
	buf.WriteString("---\n")
	buf.WriteString("title: Performance\n")
	buf.WriteString("weight: 6\n")
	buf.WriteString("---\n\n")
	buf.WriteString("Gaze includes a small benchmark suite. The point is to watch trend lines across platforms, not to pretend one machine tells the whole story.\n\n")
	buf.WriteString("Run the local suite with:\n\n")
	buf.WriteString("```sh\n")
	buf.WriteString("go test ./... -run=^$ -bench=. -benchmem\n")
	buf.WriteString("```\n\n")
	buf.WriteString("The committed numbers below are generated from the `Benchmarks` GitHub Actions workflow. That workflow runs the same benchmark suite on Linux, macOS, and Windows, then rewrites this page with the latest published results.\n\n")

	if len(docs) == 0 {
		buf.WriteString("No published benchmark data is committed yet.\n")
	} else {
		buf.WriteString("## Latest published results\n\n")
		for _, doc := range docs {
			fmt.Fprintf(&buf, "### `%s/%s`\n\n", doc.GOOS, doc.GOARCH)
			if doc.CPU != "" {
				fmt.Fprintf(&buf, "CPU: `%s`\n\n", doc.CPU)
			}
			buf.WriteString("| Benchmark | ns/op | B/op | allocs/op |\n")
			buf.WriteString("| --- | ---: | ---: | ---: |\n")
			for _, row := range doc.Benchmarks {
				fmt.Fprintf(&buf, "| `%s` | `%s` | `%s` | `%s` |\n", row.Name, row.NsPerOp, row.BPerOp, row.AllocsOp)
			}
			buf.WriteString("\n")
		}
	}

	buf.WriteString("How to read this page:\n\n")
	buf.WriteString("- `BenchmarkWatchDirectoryCreateRemove` is closest to real watcher work. It includes filesystem activity and backend event handling, so it is not a pure microbenchmark.\n")
	buf.WriteString("- `BenchmarkOpString`, `BenchmarkFilterShouldExclude`, and `BenchmarkTreeMatches` are tighter hot-path checks.\n")
	buf.WriteString("- Absolute numbers will move with hardware, runner class, Go version, and filesystem behavior. The useful signal is whether a change moves runtime, allocations, or both.\n")
	buf.WriteString("- If you want fresh committed numbers, run the `Benchmarks` workflow in GitHub Actions.\n")

	if err := os.WriteFile(outPath, buf.Bytes(), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "write %s: %v\n", outPath, err)
		os.Exit(1)
	}
}

func parseBenchmarkFile(path string) (benchmarkDoc, error) {
	file, err := os.Open(path)
	if err != nil {
		return benchmarkDoc{}, err
	}
	defer func() {
		_ = file.Close()
	}()

	var doc benchmarkDoc

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		switch {
		case strings.HasPrefix(line, "goos: "):
			doc.GOOS = strings.TrimPrefix(line, "goos: ")
		case strings.HasPrefix(line, "goarch: "):
			doc.GOARCH = strings.TrimPrefix(line, "goarch: ")
		case strings.HasPrefix(line, "cpu: "):
			doc.CPU = strings.TrimPrefix(line, "cpu: ")
		case strings.HasPrefix(line, "Benchmark"):
			row, ok := parseBenchmarkRow(line)
			if ok {
				doc.Benchmarks = append(doc.Benchmarks, row)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return benchmarkDoc{}, err
	}

	if doc.GOOS == "" || doc.GOARCH == "" {
		return benchmarkDoc{}, fmt.Errorf("missing goos/goarch metadata")
	}
	if len(doc.Benchmarks) == 0 {
		return benchmarkDoc{}, fmt.Errorf("no benchmark rows found")
	}

	return doc, nil
}

func parseBenchmarkRow(line string) (benchmarkRow, bool) {
	fields := strings.Fields(line)
	if len(fields) < 4 {
		return benchmarkRow{}, false
	}
	if fields[3] != "ns/op" {
		return benchmarkRow{}, false
	}

	row := benchmarkRow{
		Name:    trimBenchmarkSuffix(fields[0]),
		NsPerOp: fields[2] + " " + fields[3],
	}

	for i := 4; i+1 < len(fields); i += 2 {
		switch fields[i+1] {
		case "B/op":
			row.BPerOp = fields[i] + " " + fields[i+1]
		case "allocs/op":
			row.AllocsOp = fields[i] + " " + fields[i+1]
		}
	}

	if row.BPerOp == "" {
		row.BPerOp = "-"
	}
	if row.AllocsOp == "" {
		row.AllocsOp = "-"
	}

	return row, true
}

func trimBenchmarkSuffix(name string) string {
	if idx := strings.LastIndexByte(name, '-'); idx != -1 {
		return name[:idx]
	}
	return name
}
