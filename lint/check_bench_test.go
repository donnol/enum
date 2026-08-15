package enumlint

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// BenchmarkSyntheticModule measures scan+report over a synthetic module with
// many named-type enums and writes. This exercises the type index
// (namedStructFields) and the enum index (write detection), which were
// optimized from linear scans to O(1) lookups.
func BenchmarkSyntheticModule(b *testing.B) {
	dir := b.TempDir()
	const (
		nFiles  = 200
		perFile = 10
	)
	pendingTag := "`enum:\"pending,待处理\"`"
	activeTag := "`enum:\"active,活跃\"`"
	for i := 0; i < nFiles; i++ {
		var sb strings.Builder
		sb.WriteString("package synth\n\nimport \"github.com/donnol/enum\"\n\n")
		for j := 0; j < perFile; j++ {
			fmt.Fprintf(&sb, `
type Status%d_%d string
type Statuses%d_%d struct {
	enum.Struct[Status%d_%d]
	Pending Status%d_%d %s
	Active  Status%d_%d %s
}
var All%d_%d = enum.InitFor[Status%d_%d, Statuses%d_%d]()
func write%d_%d() { All%d_%d.Active = "hacked" }
`, i, j, i, j, i, j, i, j, pendingTag, i, j, activeTag, i, j, i, j, i, j, i, j, i, j)
		}
		if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("f%d.go", i)), []byte(sb.String()), 0o644); err != nil {
			b.Fatal(err)
		}
	}

	for b.Loop() {
		c := newCheckerWithModulePath([]string{dir}, "github.com/donnol/enum")
		if err := c.scan(); err != nil {
			b.Fatal(err)
		}
		if got := c.report(); len(got) == 0 {
			b.Fatal("expected violations")
		}
	}
}
