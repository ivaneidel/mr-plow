//go:build ignore

// Regenerates docs/04-CATALOG.md:  go run scripts/catalogdoc.go > docs/04-CATALOG.md
package main

import (
	"fmt"
	"strings"

	"github.com/ivaneidel/mr-plow/internal/catalog"
)

var modes = []string{"rm -rf", "empty folder", "expand children", "native command", "info only"}
var safeties = []string{"Safe", "Regenerable", "Review", "Personal"}

func main() {
	fmt.Println("# 04 · Catalog")
	fmt.Println()
	fmt.Println("Generated from `internal/catalog` by `go run scripts/catalogdoc.go > docs/04-CATALOG.md`. Don't edit by hand.")
	fmt.Println()
	fmt.Println("**Modes**: *rm -rf* removes the path · *empty folder* removes what's inside and keeps the folder · *expand children* offers each child above a size threshold as its own item · *native command* runs the tool's own cleanup · *info only* is shown but never acted on.")
	for _, os := range []string{"darwin", "linux"} {
		fmt.Printf("\n## %s\n\n| location | group | mode | safety | label | note |\n|---|---|---|---|---|---|\n", os)
		for _, e := range catalog.For(os) {
			mode := modes[e.Mode]
			saf := safeties[e.Safety]
			if e.Mode == catalog.ModeExpand {
				saf = "children: " + safeties[e.ChildSafety]
			}
			if e.Mode == catalog.ModeNative {
				mode = "`" + strings.Join(e.Native, " ") + "`"
			}
			p := e.Path
			if p == "" {
				p = "(Time Machine local snapshots)"
			}
			fmt.Printf("| `%s` | %s | %s | %s | %s | %s |\n", p, e.Group, mode, saf, e.Label, strings.ReplaceAll(e.Explain.EN, "|", "\\|"))
		}
	}
	fmt.Println("\n## Project-artifact detectors\n\nChecked for every directory during the walk. Nested matches (a `node_modules` inside a `node_modules`) are not tagged, and neither are matches inside tool-managed areas (any hidden folder under home, `~/Library`, `~/Applications`).\n\n| id | directory names | requires a sibling | safety |\n|---|---|---|---|")
	for _, d := range catalog.Detectors {
		sib := "—"
		if len(d.Siblings) > 0 {
			sib = "`" + strings.Join(d.Siblings, "`, `") + "`"
		}
		fmt.Printf("| %s | `%s` | %s | %s |\n", d.ID, strings.Join(d.Names, "`, `"), sib, safeties[d.Safety])
	}
	fmt.Println("\n## Old big files\n\nFiles from the top-500 heap (≥ 50 MB) not modified for a year, outside tool-managed areas, that are disk images, archives/installers or videos, or are over 1 GB.")
}
