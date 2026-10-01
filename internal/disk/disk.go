// Package disk reports volume usage.
package disk

import "golang.org/x/sys/unix"

// Usage of the volume holding a path.
type Usage struct {
	Mount              string
	Total, Free, Avail int64
}

// Used is what's not available to you (other APFS volumes included).
func (u Usage) Used() int64 { return u.Total - u.Avail }

// Pct is used/total in 0..1.
func (u Usage) Pct() float64 {
	if u.Total == 0 {
		return 0
	}
	return float64(u.Used()) / float64(u.Total)
}

// Of returns usage for the filesystem containing path.
func Of(path string) (Usage, error) {
	var st unix.Statfs_t
	if err := unix.Statfs(path, &st); err != nil {
		return Usage{}, err
	}
	bs := int64(st.Bsize)
	return Usage{
		Mount: mountName(&st, path),
		Total: int64(st.Blocks) * bs,
		Free:  int64(st.Bfree) * bs,
		Avail: int64(st.Bavail) * bs,
	}, nil
}
