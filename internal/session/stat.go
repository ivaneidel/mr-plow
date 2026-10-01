package session

import "golang.org/x/sys/unix"

type scanStat struct{ dev uint64 }

func statDev(p string, out *scanStat) error {
	var st unix.Stat_t
	if err := unix.Stat(p, &st); err != nil {
		return err
	}
	out.dev = uint64(st.Dev)
	return nil
}
