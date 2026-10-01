package disk

import "golang.org/x/sys/unix"

func mountName(st *unix.Statfs_t, _ string) string {
	return unix.ByteSliceToString(st.Mntonname[:])
}
