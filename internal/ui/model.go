// Package ui is the Bubble Tea terminal interface.
package ui

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ivaneidel/mr-plow/internal/apply"
	"github.com/ivaneidel/mr-plow/internal/catalog"
	"github.com/ivaneidel/mr-plow/internal/disk"
	"github.com/ivaneidel/mr-plow/internal/i18n"
	"github.com/ivaneidel/mr-plow/internal/plan"
	"github.com/ivaneidel/mr-plow/internal/scan"
	"github.com/ivaneidel/mr-plow/internal/session"
)

// Options for the UI.
type Options struct {
	DryRun    bool   // Apply is disabled; only the script can be exported
	ScriptOut string // where `e` writes the plan
}

type screen int

const (
	scMain screen = iota
	scDryRun
	scApplying
	scReport
)

type tab int

const (
	tabSuspects tab = iota
	tabExplore
	tabBig
)

type tickMsg time.Time

type applyStepMsg struct {
	i int
	r *apply.Result
}

type applyDoneMsg struct {
	results []apply.Result
	err     error
}

// Model is the whole UI state.
type Model struct {
	sess *session.Session
	opts Options
	w, h int
	tick int

	v        session.View
	lastRoot *scan.Dir

	screen screen
	tab    tab
	help   bool
	flash  string
	flashT time.Time

	selected map[string]*session.Item

	// suspects
	sKey       string // cursor anchored to a row key so it survives re-sorting
	sCursor    int
	sOffset    int
	expanded   map[string]bool
	openG      map[catalog.Group]bool // groups start collapsed

	// explore
	exNode   *scan.Dir
	exCursor int
	exOffset int
	exHist   []int
	exFiles  map[string][]scan.Entry

	// big files
	bCursor, bOffset int

	// dry run
	plan        *plan.Plan
	dCursor     int
	dOffset     int
	confirm     bool
	confirmText string

	// apply
	applyCh     chan tea.Msg
	applyIdx    int
	results     []apply.Result
	applyErr    error
	diskBefore  disk.Usage
	diskAfter   disk.Usage
	sudoScript  string
	cancel      context.CancelFunc
	scanWasDone bool
}

// New creates the UI model for a started session.
func New(sess *session.Session, opts Options) *Model {
	m := &Model{
		sess: sess, opts: opts,
		selected:   map[string]*session.Item{},
		expanded:   map[string]bool{},
		openG:      map[catalog.Group]bool{},
		exFiles:    map[string][]scan.Entry{},
	}
	m.v = sess.View()
	m.lastRoot = m.v.Snap.Root
	m.exNode = m.v.Snap.Root
	return m
}

// Run starts the program.
func Run(sess *session.Session, opts Options) error {
	_, err := tea.NewProgram(New(sess, opts), tea.WithAltScreen()).Run()
	return err
}

func tickCmd() tea.Cmd {
	return tea.Tick(200*time.Millisecond, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func (m *Model) Init() tea.Cmd { return tickCmd() }

func (m *Model) setFlash(s string) { m.flash, m.flashT = s, time.Now() }

func (m *Model) refresh() {
	m.v = m.sess.View()
	if r := m.v.Snap.Root; r != m.lastRoot {
		// the fresh scan replaced the cached one: re-anchor pointers by path
		if m.exNode != nil {
			if n := r.Lookup(m.exNode.Path()); n != nil {
				m.exNode = n
			} else {
				m.exNode, m.exHist = r, nil
			}
		} else {
			m.exNode = r
		}
		m.exFiles = map[string][]scan.Entry{}
		m.remapSelection()
		m.lastRoot = r
	}
}

// remapSelection points selected items at the current tree.
func (m *Model) remapSelection() {
	byKey := map[string]*session.Item{}
	for _, g := range m.v.Groups {
		for _, it := range g.Items {
			byKey[it.Key] = it
			for _, k := range it.Kids {
				byKey[k.Key] = k
			}
		}
	}
	for k, it := range m.selected {
		if n := byKey[k]; n != nil {
			m.selected[k] = n
			continue
		}
		if it.Node != nil {
			if n := m.v.Snap.Root.Lookup(it.Path); n != nil {
				it.Node = n
			}
		}
	}
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
		return m, nil
	case tickMsg:
		m.tick++
		if m.screen == scMain || m.screen == scDryRun {
			m.refresh()
		}
		if m.flash != "" && time.Since(m.flashT) > 4*time.Second {
			m.flash = ""
		}
		return m, tickCmd()
	case applyStepMsg:
		m.applyIdx = msg.i
		if msg.r != nil {
			m.results = append(m.results, *msg.r)
		}
		return m, m.waitApply()
	case applyDoneMsg:
		m.finishApply(msg)
		return m, nil
	case tea.KeyMsg:
		return m.key(msg)
	}
	return m, nil
}

func (m *Model) key(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	s := k.String()
	if s == "ctrl+c" {
		if m.cancel != nil {
			m.cancel()
		}
		m.sess.Stop()
		return m, tea.Quit
	}
	if m.help {
		m.help = false
		return m, nil
	}
	switch m.screen {
	case scDryRun:
		return m.keyDryRun(k)
	case scApplying:
		return m, nil
	case scReport:
		switch s {
		case "q":
			m.sess.Stop()
			return m, tea.Quit
		case "enter", "esc", " ":
			m.screen = scMain
			m.refresh()
		}
		return m, nil
	}

	switch s {
	case "q":
		m.sess.Stop()
		return m, tea.Quit
	case "?":
		m.help = true
	case "tab":
		m.tab = (m.tab + 1) % 3
	case "shift+tab":
		m.tab = (m.tab + 2) % 3
	case "1":
		m.tab = tabSuspects
	case "2":
		m.tab = tabExplore
	case "3":
		m.tab = tabBig
	case "L":
		i18n.Toggle()
	case "r":
		if err := m.sess.Rescan(); err == nil {
			m.refresh()
			m.setFlash(T("flash.rescan"))
		}
	case "c":
		m.selected = map[string]*session.Item{}
	case "d", "delete":
		m.openDryRun()
	default:
		switch m.tab {
		case tabSuspects:
			m.keySuspects(s)
		case tabExplore:
			m.keyExplore(s)
		case tabBig:
			m.keyBig(s)
		}
	}
	return m, nil
}

// --- selection ---------------------------------------------------------------

func (m *Model) isSelected(it *session.Item) bool {
	if len(it.Kids) > 0 {
		for _, k := range it.Kids {
			if m.selected[k.Key] == nil {
				return false
			}
		}
		return true
	}
	return m.selected[it.Key] != nil
}

func (m *Model) partlySelected(it *session.Item) bool {
	for _, k := range it.Kids {
		if m.selected[k.Key] != nil {
			return true
		}
	}
	return false
}

func (m *Model) toggle(it *session.Item) {
	if it == nil {
		return
	}
	if !it.Actionable() {
		m.setFlash(T("flash.info_only"))
		return
	}
	if len(it.Kids) > 0 {
		on := !m.isSelected(it)
		for _, k := range it.Kids {
			if on {
				m.selected[k.Key] = k
			} else {
				delete(m.selected, k.Key)
			}
		}
		return
	}
	if m.selected[it.Key] != nil {
		delete(m.selected, it.Key)
	} else {
		m.selected[it.Key] = it
	}
}

func (m *Model) selectionList() []*session.Item {
	out := make([]*session.Item, 0, len(m.selected))
	for _, it := range m.selected {
		out = append(out, it)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out
}

func (m *Model) selectionSize() int64 { return session.UnionSize(m.selectionList()) }

// --- dry run & apply -----------------------------------------------------------

func (m *Model) openDryRun() {
	if len(m.selected) == 0 {
		m.setFlash(T("flash.nothing_selected"))
		return
	}
	m.plan = plan.Build(m.selectionList(), plan.Context{Home: m.sess.Home, Root: m.sess.Root, Root0: m.sess.ID.Root})
	m.dCursor, m.dOffset, m.confirm, m.confirmText = 0, 0, false, ""
	m.screen = scDryRun
}

func (m *Model) keyDryRun(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	s := k.String()
	if m.confirm {
		if m.sess.ID.Root {
			switch k.Type {
			case tea.KeyEnter:
				if strings.TrimSpace(m.confirmText) == "yes" {
					return m, m.startApply()
				}
				m.confirm, m.confirmText = false, ""
			case tea.KeyEsc:
				m.confirm, m.confirmText = false, ""
			case tea.KeyBackspace:
				if m.confirmText != "" {
					m.confirmText = m.confirmText[:len(m.confirmText)-1]
				}
			case tea.KeyRunes:
				m.confirmText += string(k.Runes)
			}
			return m, nil
		}
		if s == "y" || s == "Y" {
			return m, m.startApply()
		}
		m.confirm = false
		return m, nil
	}
	n := len(m.plan.Actions)
	switch s {
	case "esc", "q", "backspace":
		m.screen = scMain
	case "up", "k":
		if m.dCursor > 0 {
			m.dCursor--
		}
	case "down", "j":
		if m.dCursor < n-1 {
			m.dCursor++
		}
	case "t":
		if n > 0 {
			a := m.plan.Actions[m.dCursor]
			switch a.Kind {
			case plan.Remove:
				a.SetTrash(true)
			case plan.Trash:
				a.SetTrash(false)
			default:
				m.setFlash(T("flash.no_trash"))
			}
		}
	case "x":
		if n > 0 {
			a := m.plan.Actions[m.dCursor]
			delete(m.selected, a.Item.Key)
			m.plan.Actions = append(m.plan.Actions[:m.dCursor], m.plan.Actions[m.dCursor+1:]...)
			if m.dCursor >= len(m.plan.Actions) && m.dCursor > 0 {
				m.dCursor--
			}
			if len(m.plan.Actions) == 0 {
				m.screen = scMain
			}
		}
	case "e":
		m.exportScript()
	case "a", "enter":
		ready, _ := m.plan.Count(plan.Ready)
		switch {
		case m.opts.DryRun:
			m.setFlash(T("flash.dry_run"))
		case ready == 0:
			m.setFlash(T("flash.nothing_ready"))
		default:
			m.confirm = true
		}
	}
	return m, nil
}

func (m *Model) exportScript() {
	path := m.opts.ScriptOut
	if path == "" {
		path = "mr-plow-plan.sh"
	}
	if err := os.WriteFile(path, []byte(m.plan.Script(nil)), 0o644); err != nil {
		m.setFlash(T("flash.export_fail") + ": " + err.Error())
		return
	}
	m.sess.ID.Chown(path)
	abs, _ := filepath.Abs(path)
	m.setFlash(T("flash.exported") + " " + abs)
}

func (m *Model) startApply() tea.Cmd {
	m.confirm = false
	m.screen = scApplying
	m.results = nil
	m.applyIdx = 0
	m.diskBefore, _ = disk.Of(m.sess.Root)
	m.scanWasDone = m.sess.ScanDone()
	m.sess.Stop() // nothing should be listing what we are deleting
	ch := make(chan tea.Msg, 16)
	m.applyCh = ch
	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel
	p := m.plan
	opts := apply.Options{ID: m.sess.ID, DryRun: m.opts.DryRun}
	go func() {
		res, err := apply.Run(ctx, p, opts, func(i int, r *apply.Result) { ch <- applyStepMsg{i, r} })
		ch <- applyDoneMsg{res, err}
	}()
	return m.waitApply()
}

func (m *Model) waitApply() tea.Cmd {
	ch := m.applyCh
	return func() tea.Msg { return <-ch }
}

func (m *Model) finishApply(msg applyDoneMsg) {
	m.results = msg.results
	m.applyErr = msg.err
	m.diskAfter, _ = disk.Of(m.sess.Root)
	for _, r := range m.results {
		if r.OK && r.Action.Kind != plan.Native {
			m.sess.Forget(r.Action.Item)
		}
		if r.OK {
			delete(m.selected, r.Action.Item.Key)
		}
	}
	// anything that needs root goes to a script the user runs themselves
	st := plan.NeedsRoot
	if n, _ := m.plan.Count(plan.NeedsRoot); n > 0 {
		path := strings.TrimSuffix(m.opts.ScriptOut, ".sh")
		if path == "" {
			path = "mr-plow-plan"
		}
		path += "-sudo.sh"
		if os.WriteFile(path, []byte(m.plan.Script(&st)), 0o644) == nil {
			m.sess.ID.Chown(path)
			m.sudoScript, _ = filepath.Abs(path)
		}
	}
	m.screen = scReport
	m.cancel = nil
	if m.scanWasDone {
		m.sess.SaveCache()
	} else {
		_ = m.sess.Rescan() // the interrupted scan is redone; current data stays visible
	}
}

// --- suspects ------------------------------------------------------------------

type rowKind int

const (
	rowGroup rowKind = iota
	rowItem
	rowKid
)

type row struct {
	kind rowKind
	key  string
	g    *session.GroupView
	it   *session.Item
}

func (m *Model) suspectRows() []row {
	var rows []row
	for gi := range m.v.Groups {
		g := &m.v.Groups[gi]
		rows = append(rows, row{kind: rowGroup, key: "g:" + string(g.Group), g: g})
		if !m.openG[g.Group] {
			continue
		}
		for _, it := range g.Items {
			rows = append(rows, row{kind: rowItem, key: it.Key, g: g, it: it})
			if len(it.Kids) > 0 && m.expanded[it.Key] {
				for _, k := range it.Kids {
					rows = append(rows, row{kind: rowKid, key: k.Key, g: g, it: k})
				}
			}
		}
	}
	return rows
}

func (m *Model) suspectCursor(rows []row) int {
	if m.sKey != "" {
		for i, r := range rows {
			if r.key == m.sKey {
				return i
			}
		}
	}
	if m.sCursor >= len(rows) {
		m.sCursor = len(rows) - 1
	}
	if m.sCursor < 0 {
		m.sCursor = 0
	}
	return m.sCursor
}

func (m *Model) keySuspects(s string) {
	rows := m.suspectRows()
	if len(rows) == 0 {
		return
	}
	c := m.suspectCursor(rows)
	move := func(to int) {
		if to < 0 {
			to = 0
		}
		if to >= len(rows) {
			to = len(rows) - 1
		}
		m.sCursor = to
		m.sKey = rows[to].key
	}
	r := rows[c]
	switch s {
	case "up", "k":
		move(c - 1)
	case "down", "j":
		move(c + 1)
	case "pgup":
		move(c - m.listHeight())
	case "pgdown":
		move(c + m.listHeight())
	case "home", "g":
		move(0)
	case "end", "G":
		move(len(rows) - 1)
	case " ":
		if r.kind == rowGroup {
			// toggle every Safe/Regenerable item in the group; the cursor stays
			// on the group so a second space undoes the first
			var easy []*session.Item
			all := true
			for _, it := range r.g.Items {
				if it.Actionable() && (it.Safety == catalog.Safe || it.Safety == catalog.Regenerable) {
					easy = append(easy, it)
					all = all && m.isSelected(it)
				}
			}
			for _, it := range easy {
				if m.isSelected(it) == all {
					m.toggle(it)
				}
			}
			return
		}
		m.toggle(r.it)
		move(c + 1)
	case "enter", "right", "l":
		switch {
		case r.kind == rowGroup:
			m.openG[r.g.Group] = !m.openG[r.g.Group]
		case len(r.it.Kids) > 0:
			m.expanded[r.it.Key] = !m.expanded[r.it.Key]
		case r.it.Node != nil:
			m.exploreTo(r.it.Node)
		}
	case "left", "h":
		switch r.kind {
		case rowGroup:
			m.openG[r.g.Group] = false
		case rowKid:
			for i := c; i >= 0; i-- {
				if rows[i].kind == rowItem {
					m.expanded[rows[i].it.Key] = false
					move(i)
					break
				}
			}
		case rowItem:
			if m.expanded[r.it.Key] {
				m.expanded[r.it.Key] = false
			} else {
				m.openG[r.g.Group] = false
				m.sKey = "g:" + string(r.g.Group)
			}
		}
	case "o":
		if r.it != nil {
			m.reveal(r.it.Path)
		}
	}
}

// exploreTo jumps to the Explore tab focused on a node.
func (m *Model) exploreTo(d *scan.Dir) {
	m.exNode = d
	m.exCursor, m.exOffset, m.exHist = 0, 0, nil
	m.tab = tabExplore
}

func (m *Model) reveal(path string) {
	if path == "" {
		return
	}
	var cmd *exec.Cmd
	if runtime.GOOS == "darwin" {
		cmd = exec.Command("open", "-R", path)
	} else {
		cmd = exec.Command("xdg-open", filepath.Dir(path))
	}
	if cmd.Start() == nil {
		go cmd.Wait()
		m.setFlash(T("flash.revealed"))
	}
}

// --- explore -------------------------------------------------------------------

type exRow struct {
	name  string
	size  int64
	files int64
	mtime int64
	dir   *scan.Dir
	file  *scan.Entry
}

func (m *Model) exploreRows() []exRow {
	d := m.exNode
	if d == nil {
		return nil
	}
	var rows []exRow
	for _, c := range d.Children() {
		rows = append(rows, exRow{name: c.Name, size: c.Size(), files: c.Files(), mtime: c.Newest(), dir: c})
	}
	path := d.Path()
	files, ok := m.exFiles[path]
	if !ok {
		ents, _ := scan.DefaultLister().List(path, nil)
		for _, e := range ents {
			if e.Kind != scan.KindDir {
				files = append(files, e)
			}
		}
		m.exFiles[path] = files
	}
	for i := range files {
		f := &files[i]
		rows = append(rows, exRow{name: f.Name, size: f.Alloc, files: 1, mtime: f.MTime, file: f})
	}
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].size > rows[j].size })
	if len(rows) > 500 {
		rows = rows[:500]
	}
	return rows
}

func (m *Model) exploreItem(r exRow) *session.Item {
	explain := i18n.Text{EN: "Picked by hand in Explore.", ES: "Elegido a mano en Explorar."}
	parent := session.Tilde(m.exNode.Path(), m.sess.Home)
	if r.dir != nil {
		p := r.dir.Path()
		return &session.Item{Key: "e:" + p, Label: r.name + "/", Sub: parent, Explain: explain,
			Safety: catalog.Review, Path: p, Mode: catalog.ModeRemove, Node: r.dir}
	}
	p := filepath.Join(m.exNode.Path(), r.name)
	f := &scan.File{Path: p, Size: r.file.Alloc, MTime: r.file.MTime, Dev: r.file.Dev, Ino: r.file.Ino}
	return &session.Item{Key: "f:" + p, Label: r.name, Sub: parent, Explain: explain,
		Safety: catalog.Review, Path: p, Mode: catalog.ModeRemove, File: f, FileDir: m.exNode}
}

func (m *Model) keyExplore(s string) {
	rows := m.exploreRows()
	n := len(rows)
	switch s {
	case "up", "k":
		if m.exCursor > 0 {
			m.exCursor--
		}
	case "down", "j":
		if m.exCursor < n-1 {
			m.exCursor++
		}
	case "pgup":
		m.exCursor = max(0, m.exCursor-m.listHeight())
	case "pgdown":
		m.exCursor = min(n-1, m.exCursor+m.listHeight())
	case "home", "g":
		m.exCursor = 0
	case "end", "G":
		m.exCursor = n - 1
	case "enter", "right", "l":
		if m.exCursor < n && rows[m.exCursor].dir != nil {
			d := rows[m.exCursor].dir
			if d.Has(scan.FlagDenied) {
				m.setFlash(T("flash.denied"))
				return
			}
			m.exHist = append(m.exHist, m.exCursor)
			m.exNode = d
			m.exCursor, m.exOffset = 0, 0
		}
	case "left", "h", "backspace":
		if m.exNode != nil && m.exNode.Parent != nil {
			child := m.exNode
			m.exNode = m.exNode.Parent
			m.exCursor, m.exOffset = 0, 0
			if k := len(m.exHist); k > 0 {
				m.exCursor = m.exHist[k-1]
				m.exHist = m.exHist[:k-1]
			}
			for i, r := range m.exploreRows() {
				if r.dir == child {
					m.exCursor = i
				}
			}
		}
	case " ":
		if m.exCursor < n {
			m.toggle(m.exploreItem(rows[m.exCursor]))
			if m.exCursor < n-1 {
				m.exCursor++
			}
		}
	case "o":
		if m.exCursor < n {
			m.reveal(filepath.Join(m.exNode.Path(), rows[m.exCursor].name))
		}
	}
	if m.exCursor < 0 {
		m.exCursor = 0
	}
}

// --- big files -----------------------------------------------------------------

func (m *Model) bigItem(f *scan.File) *session.Item {
	if it := m.selected["f:"+f.Path]; it != nil {
		return it
	}
	return &session.Item{Key: "f:" + f.Path, Label: filepath.Base(f.Path), Sub: session.Tilde(filepath.Dir(f.Path), m.sess.Home),
		Explain: i18n.Text{EN: "One of the biggest files on this disk.", ES: "Uno de los archivos más grandes del disco."},
		Safety:  catalog.Review, Path: f.Path, Mode: catalog.ModeRemove, File: f,
		FileDir: m.v.Snap.Root.Lookup(filepath.Dir(f.Path))}
}

func (m *Model) keyBig(s string) {
	n := len(m.v.Snap.Big)
	switch s {
	case "up", "k":
		if m.bCursor > 0 {
			m.bCursor--
		}
	case "down", "j":
		if m.bCursor < n-1 {
			m.bCursor++
		}
	case "pgup":
		m.bCursor = max(0, m.bCursor-m.listHeight())
	case "pgdown":
		m.bCursor = min(n-1, m.bCursor+m.listHeight())
	case "home", "g":
		m.bCursor = 0
	case "end", "G":
		m.bCursor = n - 1
	case " ":
		if m.bCursor < n {
			m.toggle(m.bigItem(&m.v.Snap.Big[m.bCursor]))
			if m.bCursor < n-1 {
				m.bCursor++
			}
		}
	case "o":
		if m.bCursor < n {
			m.reveal(m.v.Snap.Big[m.bCursor].Path)
		}
	case "enter", "right", "l":
		if m.bCursor < n {
			if d := m.v.Snap.Root.Lookup(filepath.Dir(m.v.Snap.Big[m.bCursor].Path)); d != nil {
				m.exploreTo(d)
			}
		}
	}
	if m.bCursor < 0 {
		m.bCursor = 0
	}
}
