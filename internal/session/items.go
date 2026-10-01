package session

import (
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/ivaneidel/mr-plow/internal/catalog"
	"github.com/ivaneidel/mr-plow/internal/i18n"
	"github.com/ivaneidel/mr-plow/internal/scan"
)

// Item is one row on the Suspects screen: something you could remove.
type Item struct {
	Key     string
	Group   catalog.Group
	Label   string
	Sub     string // secondary text (project path, parent folder…)
	Explain i18n.Text
	Safety  catalog.Safety
	Path    string
	Mode    catalog.Mode // Remove, Contents, Native or Info
	Native  []string
	System  bool // outside home: usually needs root
	Node    *scan.Dir
	File    *scan.File
	FileDir *scan.Dir // directory node holding File (to update sizes after delete)
	Kids    []*Item   // aggregated detector instances
	Unknown bool      // size can't be measured (e.g. snapshots)
	Count   int       // instances (detector folders, snapshots); 0 = n/a
	Cached  bool      // comes from a cached snapshot
}

// Size is the item's current size (live while scanning).
func (it *Item) Size() int64 {
	switch {
	case it.Node != nil:
		return it.Node.Size()
	case it.File != nil:
		return it.File.Size
	case len(it.Kids) > 0:
		var n int64
		for _, k := range it.Kids {
			n += k.Size()
		}
		return n
	}
	return 0
}

// Newest is the newest mtime inside (0 if unknown yet).
func (it *Item) Newest() int64 {
	switch {
	case it.Node != nil:
		return it.Node.Newest()
	case it.File != nil:
		return it.File.MTime
	case len(it.Kids) > 0:
		var n int64
		for _, k := range it.Kids {
			if v := k.Newest(); v > n {
				n = v
			}
		}
		return n
	}
	return 0
}

// Done reports whether the size is final.
func (it *Item) Done() bool {
	switch {
	case it.Node != nil:
		return it.Node.Done()
	case len(it.Kids) > 0:
		for _, k := range it.Kids {
			if !k.Done() {
				return false
			}
		}
	}
	return true
}

// Actionable reports whether the item can go into a plan.
func (it *Item) Actionable() bool { return it.Mode != catalog.ModeInfo }

// GroupView is a Suspects section with its items, biggest first.
type GroupView struct {
	Group catalog.Group
	Items []*Item
	Total int64 // without double-counting nested paths
}

// Tilde shortens a path under home to ~/…
func Tilde(path, home string) string {
	if home != "" && (path == home || strings.HasPrefix(path, home+"/")) {
		return "~" + path[len(home):]
	}
	return path
}

func under(path, dir string) bool { return path == dir || strings.HasPrefix(path, dir+"/") }

// buildItems turns the catalog + snapshots into Suspects groups.
func (s *Session) buildItems(main *scan.Snapshot) []GroupView {
	var items []*Item
	lookup := func(p string) *scan.Dir {
		if main != nil {
			if d := main.Root.Lookup(p); d != nil {
				return d
			}
		}
		for root, sn := range s.extraSnaps() {
			if under(p, root) {
				return sn.Root.Lookup(p)
			}
		}
		return nil
	}

	// explicit (non-expand) entry paths: expanded children equal to them are skipped
	explicit := map[string]catalog.Entry{}
	for _, e := range s.entries {
		if e.Mode == catalog.ModeExpand {
			continue
		}
		for _, p := range e.Resolve(s.Home) {
			explicit[p] = e
		}
	}
	// expanded children that contain a tool-managed location (a native-command
	// entry or another expanded entry, e.g. UTM's VMs inside ~/Library/Containers)
	// are left to that entry, so the same bytes aren't offered twice.
	var managed []string
	for _, e := range s.entries {
		if e.Mode != catalog.ModeNative && e.Mode != catalog.ModeExpand {
			continue
		}
		managed = append(managed, e.Resolve(s.Home)...)
	}
	containsManaged := func(p string) bool {
		for _, mp := range managed {
			if strings.HasPrefix(mp, p+"/") {
				return true
			}
		}
		return false
	}
	cached := main != nil && main.Cached

	for _, e := range s.entries {
		if e.ID == "tm-snapshots" {
			if n := s.tmSnapshots.Load(); n > 0 {
				items = append(items, &Item{Key: "tm-snapshots", Group: e.Group, Label: e.Label, Explain: e.Explain,
					Safety: e.Safety, Mode: e.Mode, Native: e.Native, System: true, Unknown: true,
					Count: int(n)})
			}
			continue
		}
		for _, p := range e.Resolve(s.Home) {
			d := lookup(p)
			if d == nil {
				continue
			}
			if e.Mode != catalog.ModeExpand {
				if d.Done() && d.Size() < 1<<20 {
					continue // nothing worth showing (e.g. an empty Trash)
				}
				it := &Item{Key: e.ID + ":" + p, Group: e.Group, Label: e.Label, Explain: e.Explain, Safety: e.Safety,
					Path: p, Mode: e.Mode, Native: e.Native, System: e.System, Node: d, Cached: cached}
				if strings.ContainsAny(e.Path, "*?[") {
					it.Sub = Tilde(filepath.Dir(p), s.Home)
					it.Label = e.Label + " · " + filepath.Base(p)
				}
				items = append(items, it)
				continue
			}
			min := e.ExpandMin
			if min == 0 {
				min = 50 << 20
			}
			depth := e.ExpandDepth
			if depth == 0 {
				depth = 1
			}
			level := []*scan.Dir{d}
			for i := 0; i < depth; i++ {
				var next []*scan.Dir
				for _, n := range level {
					next = append(next, n.Children()...)
				}
				level = next
			}
			mode := e.ChildMode
			if mode == catalog.ModeExpand {
				mode = catalog.ModeRemove
			}
			for _, c := range level {
				cp := c.Path()
				if _, ok := explicit[cp]; ok || containsManaged(cp) || c.Size() < min || c.Has(scan.FlagMount) {
					continue
				}
				label := c.Name
				if depth > 1 {
					label = strings.TrimPrefix(cp, p+"/") // e.g. nodejs/22.16.0
				}
				items = append(items, &Item{Key: "x:" + cp, Group: e.Group, Label: label, Sub: e.Label,
					Explain: e.Explain, Safety: e.ChildSafety, Path: cp, Mode: mode, System: e.System, Node: c, Cached: cached})
			}
			// big files sitting directly at the expansion level
			if main != nil && depth == 1 {
				for i := range main.Big {
					f := &main.Big[i]
					if filepath.Dir(f.Path) == p && f.Size >= min {
						items = append(items, &Item{Key: "f:" + f.Path, Group: e.Group, Label: filepath.Base(f.Path), Sub: e.Label,
							Explain: e.Explain, Safety: e.ChildSafety, Path: f.Path, Mode: catalog.ModeRemove, File: f,
							FileDir: d, Cached: cached})
					}
				}
			}
		}
	}

	if main != nil {
		items = append(items, s.detectorItems(main)...)
		items = append(items, s.oldBigFiles(main, items)...)
	}
	return groupItems(items)
}

// toolManaged reports paths inside tool/app-managed areas (hidden dirs under
// home, ~/Library…), where a node_modules belongs to a tool, not a project.
func toolManaged(path, home string) bool {
	rel := strings.TrimPrefix(Tilde(path, home), "~/")
	if rel == path {
		return false // outside home: judge by the scan root alone
	}
	parts := strings.Split(rel, "/")
	for _, p := range parts[:len(parts)-1] {
		if strings.HasPrefix(p, ".") || p == "Library" || p == "Applications" {
			return true
		}
	}
	return false
}

func (s *Session) detectorItems(main *scan.Snapshot) []*Item {
	byID := map[string]*Item{}
	var order []string
	for _, d := range main.Tagged {
		if d.Has(scan.FlagRemoved) {
			continue
		}
		p := d.Path()
		if toolManaged(p, s.Home) {
			continue
		}
		det := catalog.DetectorByID(d.Tag)
		if det == nil {
			continue
		}
		agg := byID[det.ID]
		if agg == nil {
			agg = &Item{Key: "det:" + det.ID, Group: catalog.GroupDev, Label: det.Label, Explain: det.Explain, Safety: det.Safety, Mode: catalog.ModeRemove}
			byID[det.ID] = agg
			order = append(order, det.ID)
		}
		agg.Kids = append(agg.Kids, &Item{Key: "d:" + p, Group: catalog.GroupDev, Label: Tilde(filepath.Dir(p), s.Home),
			Sub: det.Label, Explain: det.Explain, Safety: det.Safety, Path: p, Mode: catalog.ModeRemove, Node: d, Cached: main.Cached})
	}
	var out []*Item
	for _, id := range order {
		agg := byID[id]
		sort.Slice(agg.Kids, func(i, j int) bool { return agg.Kids[i].Size() > agg.Kids[j].Size() })
		agg.Count = len(agg.Kids)
		out = append(out, agg)
	}
	return out
}

var oldExts = map[string]bool{
	".iso": true, ".dmg": true, ".img": true, ".vmdk": true, ".vdi": true, ".qcow2": true, ".vhd": true, ".vhdx": true,
	".zip": true, ".tar": true, ".gz": true, ".tgz": true, ".7z": true, ".rar": true, ".xip": true, ".pkg": true, ".ipa": true, ".apk": true,
	".mp4": true, ".mov": true, ".mkv": true, ".avi": true, ".wmv": true, ".m4v": true,
}

// oldBigFiles: big files untouched for a year that look like images,
// archives, installers or videos (or are just huge), outside app-managed areas.
func (s *Session) oldBigFiles(main *scan.Snapshot, already []*Item) []*Item {
	seen := map[string]bool{}
	for _, it := range already {
		if it.File != nil {
			seen[it.Path] = true
		}
	}
	cut := time.Now().AddDate(-1, 0, 0).Unix()
	var out []*Item
	for i := range main.Big {
		f := &main.Big[i]
		if seen[f.Path] || f.MTime > cut || toolManaged(f.Path, s.Home) {
			continue
		}
		ext := strings.ToLower(filepath.Ext(f.Path))
		if !oldExts[ext] && f.Size < 1<<30 {
			continue
		}
		out = append(out, &Item{Key: "f:" + f.Path, Group: catalog.GroupBig, Label: filepath.Base(f.Path),
			Sub: Tilde(filepath.Dir(f.Path), s.Home), Explain: fileKind(ext), Safety: catalog.Review, Path: f.Path,
			Mode: catalog.ModeRemove, File: f, FileDir: main.Root.Lookup(filepath.Dir(f.Path)), Cached: main.Cached})
	}
	return out
}

func fileKind(ext string) i18n.Text {
	switch ext {
	case ".iso", ".dmg", ".img", ".vmdk", ".vdi", ".qcow2", ".vhd", ".vhdx":
		return i18n.Text{EN: "Disk image untouched for over a year. Installers and old OS images are rarely needed again.",
			ES: "Imagen de disco sin tocar hace más de un año. Instaladores e imágenes de SO viejas rara vez se vuelven a usar."}
	case ".zip", ".tar", ".gz", ".tgz", ".7z", ".rar", ".xip", ".pkg", ".ipa", ".apk":
		return i18n.Text{EN: "Archive or installer untouched for over a year. Check it was extracted or installed already.",
			ES: "Comprimido o instalador sin tocar hace más de un año. Fijate si ya lo extrajiste o instalaste."}
	case ".mp4", ".mov", ".mkv", ".avi", ".wmv", ".m4v":
		return i18n.Text{EN: "Video untouched for over a year.", ES: "Video sin tocar hace más de un año."}
	}
	return i18n.Text{EN: "Huge file untouched for over a year.", ES: "Archivo enorme sin tocar hace más de un año."}
}

func groupItems(items []*Item) []GroupView {
	by := map[catalog.Group][]*Item{}
	for _, it := range items {
		by[it.Group] = append(by[it.Group], it)
	}
	var out []GroupView
	for _, g := range catalog.Groups {
		its := by[g]
		if len(its) == 0 {
			continue
		}
		sort.SliceStable(its, func(i, j int) bool { return its[i].Size() > its[j].Size() })
		out = append(out, GroupView{Group: g, Items: its, Total: UnionSize(its)})
	}
	return out
}

// UnionSize sums items without counting a path nested inside another twice.
func UnionSize(items []*Item) int64 {
	var flat []*Item
	for _, it := range items {
		if len(it.Kids) > 0 {
			flat = append(flat, it.Kids...)
		} else {
			flat = append(flat, it)
		}
	}
	sort.Slice(flat, func(i, j int) bool { return len(flat[i].Path) < len(flat[j].Path) })
	var kept []string
	var total int64
outer:
	for _, it := range flat {
		if it.Path != "" {
			for _, k := range kept {
				if under(it.Path, k) {
					continue outer
				}
			}
			kept = append(kept, it.Path)
		}
		total += it.Size()
	}
	return total
}
