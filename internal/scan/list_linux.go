//go:build linux

package scan

func platformListers() []Lister { return []Lister{FstatatLister{}} }
