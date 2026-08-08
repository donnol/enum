// Command enumlint is the CLI for the enumlint package. It scans Go
// source for writes to enum values (created with enum.InitFor) and
// exits non-zero when any are found.
//
// Usage:
//
//	enumlint [flags] <dirs...>      # dirs support ./... expansion
//	enumlint -enum-pkg github.com/donnol/enum ./...
package main

import (
	"flag"
	"fmt"
	"os"

	enumlint "github.com/donnol/enum/lint"
)

func main() {
	var cfg enumlint.Config
	flag.StringVar(&cfg.EnumPkg, "enum-pkg", "",
		"explicit enum package import path (auto-discovered when empty)")
	flag.Parse()
	cfg.Roots = flag.Args()
	if len(cfg.Roots) == 0 {
		cfg.Roots = []string{"."}
	}

	report, n, err := enumlint.Run(cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, "enumlint:", err)
		os.Exit(1)
	}
	if n == 0 {
		fmt.Println("\033[32m✅ enum check is good 🌟\033[0m")
		return
	}
	fmt.Println("\033[31m🚨 enum check is bad 💥\033[0m")
	fmt.Println(report)
	fmt.Println("\033[33m\u26A0\ufe0f  请修正后重试！\033[0m")
	os.Exit(1)
}
