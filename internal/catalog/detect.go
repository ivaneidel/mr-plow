package catalog

import "github.com/ivaneidel/mr-plow/internal/scan"

// Detect returns a scan.Options.Detect function backed by Detectors.
func Detect() func(siblings []scan.Entry, child string) string {
	byName := map[string][]*Detector{}
	for i := range Detectors {
		d := &Detectors[i]
		for _, n := range d.Names {
			byName[n] = append(byName[n], d)
		}
	}
	return func(siblings []scan.Entry, child string) string {
		ds := byName[child]
		if len(ds) == 0 {
			return ""
		}
		for _, d := range ds {
			if len(d.Siblings) == 0 {
				return d.ID
			}
			for _, s := range siblings {
				for _, want := range d.Siblings {
					if s.Name == want {
						return d.ID
					}
				}
			}
		}
		return ""
	}
}
