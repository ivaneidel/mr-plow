//go:build ignore

package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/ivaneidel/mr-plow/internal/scan"
)

func main() {
	for _, w := range os.Args[2:] {
		n, _ := strconv.Atoi(w)
		s, _ := scan.New(scan.Options{Root: os.Args[1], Workers: n})
		s.Start()
		s.Wait()
		p := s.Progress()
		fmt.Printf("workers=%-3d %8s %9d files\n", n, p.Elapsed.Round(1e6), p.Files)
	}
}
