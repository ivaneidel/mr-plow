//go:build ignore

// Compares per-subtree sizes between two listers: go run scripts/sizecmp.go <root> [depth]
package main

import (
	"fmt"
	"os"

	"github.com/ivaneidel/mr-plow/internal/scan"
)

func run(root string, l scan.Lister) map[string]int64 {
	s, err := scan.New(scan.Options{Root: root, Lister: l})
	if err != nil {
		panic(err)
	}
	s.Start()
	s.Wait()
	m := map[string]int64{}
	s.Root().Walk(func(d *scan.Dir) bool {
		m[d.Path()] = d.Size()
		return true
	})
	return m
}

func main() {
	a := run(os.Args[1], scan.ListerByName("getattrlistbulk"))
	b := run(os.Args[1], scan.GenericLister{})
	for p, sb := range b {
		sa := a[p]
		if d := sb - sa; d > 500<<20 || d < -(500<<20) {
			fmt.Printf("%12d %12d %12d %s\n", sa>>20, sb>>20, d>>20, p)
		}
	}
}
