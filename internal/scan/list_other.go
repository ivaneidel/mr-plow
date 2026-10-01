//go:build !darwin && !linux

package scan

func platformListers() []Lister { return nil }
