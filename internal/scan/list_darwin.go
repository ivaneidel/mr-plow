//go:build darwin

package scan

import (
	"encoding/binary"
	"sync"
	"syscall"
	"unsafe"

	"golang.org/x/sys/unix"
)

func platformListers() []Lister { return []Lister{&BulkLister{}, FstatatLister{}} }

// Constants from <sys/attr.h>.
const (
	attrBitMapCount = 5

	attrCmnName          = 0x00000001
	attrCmnDevID         = 0x00000002
	attrCmnObjType       = 0x00000008
	attrCmnModTime       = 0x00000400
	attrCmnFileID        = 0x02000000
	attrCmnError         = 0x20000000
	attrCmnReturnedAttrs = 0x80000000

	attrDirMountStatus = 0x00000004
	dirMntStatusMntPt  = 0x00000001

	attrFileLinkCount = 0x00000001
	attrFileAllocSize = 0x00000004

	vREG = 1
	vDIR = 2
	vLNK = 5
)

type attrList struct {
	bitmapCount uint16
	reserved    uint16
	commonAttr  uint32
	volAttr     uint32
	dirAttr     uint32
	fileAttr    uint32
	forkAttr    uint32
}

// BulkLister uses getattrlistbulk(2): one syscall returns names, types,
// allocated sizes, inodes and mtimes for many entries at once, instead of a
// readdir plus one stat per entry.
type BulkLister struct {
	pool sync.Pool
}

func (*BulkLister) Name() string { return "getattrlistbulk" }

var bulkAttrs = attrList{
	bitmapCount: attrBitMapCount,
	commonAttr: attrCmnReturnedAttrs | attrCmnError | attrCmnName | attrCmnDevID |
		attrCmnObjType | attrCmnModTime | attrCmnFileID,
	dirAttr:  attrDirMountStatus,
	fileAttr: attrFileLinkCount | attrFileAllocSize,
}

func (l *BulkLister) List(path string, dst []Entry) ([]Entry, error) {
	fd, err := unix.Open(path, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return dst, err
	}
	defer unix.Close(fd)

	bp, _ := l.pool.Get().(*[]byte)
	if bp == nil {
		b := make([]byte, 128<<10)
		bp = &b
	}
	defer l.pool.Put(bp)
	buf := *bp
	al := bulkAttrs
	le := binary.LittleEndian

	for {
		n, _, errno := syscall.Syscall6(unix.SYS_GETATTRLISTBULK, uintptr(fd),
			uintptr(unsafe.Pointer(&al)), uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)), 0, 0)
		if errno == syscall.EINTR {
			continue
		}
		if errno != 0 {
			return dst, errno
		}
		if n == 0 {
			return dst, nil
		}
		off := 0
		for i := 0; i < int(n) && off+24 <= len(buf); i++ {
			rec := buf[off:]
			length := int(le.Uint32(rec))
			if length == 0 {
				break
			}
			off += length
			retCommon := le.Uint32(rec[4:])
			retDir := le.Uint32(rec[12:])
			retFile := le.Uint32(rec[16:])
			p := 24
			if retCommon&attrCmnError != 0 {
				if le.Uint32(rec[p:]) != 0 {
					continue // entry could not be read; skip it
				}
				p += 4
			}
			var e Entry
			if retCommon&attrCmnName != 0 {
				nameOff := int(int32(le.Uint32(rec[p:])))
				nameLen := int(le.Uint32(rec[p+4:]))
				start := p + nameOff
				if nameLen > 0 && start+nameLen <= length {
					e.Name = string(rec[start : start+nameLen-1]) // drop trailing NUL
				}
				p += 8
			}
			if retCommon&attrCmnDevID != 0 {
				e.Dev = uint64(le.Uint32(rec[p:]))
				p += 4
			}
			var objType uint32
			if retCommon&attrCmnObjType != 0 {
				objType = le.Uint32(rec[p:])
				p += 4
			}
			if retCommon&attrCmnModTime != 0 {
				e.MTime = int64(le.Uint64(rec[p:]))
				p += 16
			}
			if retCommon&attrCmnFileID != 0 {
				e.Ino = le.Uint64(rec[p:])
				p += 8
			}
			if retDir&attrDirMountStatus != 0 {
				e.Mount = le.Uint32(rec[p:])&dirMntStatusMntPt != 0
				p += 4
			}
			if retFile&attrFileLinkCount != 0 {
				e.Nlink = le.Uint32(rec[p:])
				p += 4
			}
			if retFile&attrFileAllocSize != 0 {
				e.Alloc = int64(le.Uint64(rec[p:]))
			}
			switch objType {
			case vREG:
				e.Kind = KindFile
			case vDIR:
				e.Kind = KindDir
			case vLNK:
				e.Kind = KindLink
			}
			if e.Name != "" {
				dst = append(dst, e)
			}
		}
	}
}
