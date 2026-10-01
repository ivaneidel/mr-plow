package ui

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/ivaneidel/mr-plow/internal/apply"
	"github.com/ivaneidel/mr-plow/internal/catalog"
	"github.com/ivaneidel/mr-plow/internal/plan"
	"github.com/ivaneidel/mr-plow/internal/scan"
	"github.com/ivaneidel/mr-plow/internal/session"
)

const (
	headerLines = 6
	footerLines = 5
)

func (m *Model) listHeight() int {
	h := m.h - headerLines - footerLines
	if h < 3 {
		h = 3
	}
	return h
}

func (m *Model) View() string {
	if m.w == 0 {
		return ""
	}
	if m.help {
		return m.viewHelp()
	}
	var body string
	switch m.screen {
	case scDryRun:
		body = m.viewDryRun()
	case scApplying:
		body = m.viewApplying()
	case scReport:
		body = m.viewReport()
	default:
		body = m.viewMain()
	}
	return fitHeight(body, m.h)
}

func fitHeight(s string, h int) string {
	lines := strings.Split(s, "\n")
	if len(lines) > h {
		lines = lines[:h]
	}
	for len(lines) < h {
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}

// --- header --------------------------------------------------------------------

func (m *Model) header() []string {
	w := m.w
	d := m.v.Disk
	pct := d.Pct()
	gc := cSafe
	if pct > 0.9 {
		gc = cDanger
	} else if pct > 0.75 {
		gc = cReview
	}
	title := sTitle.Render("mr-plow")
	badges := ""
	if m.sess.ID.Root {
		badges += " " + sBadgeRed.Render("ROOT")
	}
	if m.opts.DryRun {
		badges += " " + sBadgeAmb.Render(T("badge.dryrun"))
	}
	diskTxt := fmt.Sprintf(" %s %s · %s %s · %s %s ", sBold.Render(human(d.Used())), T("hdr.used"),
		sBold.Render(human(d.Avail)), T("hdr.free"), T("hdr.of"), human(d.Total))
	label := sDim.Render(" " + trunc(d.Mount, 22) + " ")
	gw := w - lipgloss.Width(title) - lipgloss.Width(badges) - lipgloss.Width(diskTxt) - lipgloss.Width(label) - 7
	if gw < 10 {
		gw = 10
	}
	l1 := title + badges + label + gauge(pct, gw, gc) + fmt.Sprintf(" %3.0f%%", pct*100) + diskTxt

	// stacked bar: where the scanned root's space goes
	root := m.v.Snap.Root
	kids := root.SortedChildren()
	total := root.Size()
	bw := w - 2
	var segs, legend []string
	used := 0
	shown := 0
	for i, k := range kids {
		if i >= len(segColors) || total == 0 {
			break
		}
		cw := int(float64(k.Size()) / float64(total) * float64(bw))
		if cw < 1 {
			break
		}
		st := lipgloss.NewStyle().Foreground(segColors[i])
		segs = append(segs, st.Render(strings.Repeat("█", cw)))
		legend = append(legend, st.Render("■")+" "+trunc(k.Name, 18)+" "+sDim.Render(human(k.Size())))
		used += cw
		shown++
	}
	if used < bw {
		segs = append(segs, lipgloss.NewStyle().Foreground(cFaint).Render(strings.Repeat("█", bw-used)))
	}
	rootName := session.Tilde(root.Name, m.sess.Home)
	legendLine := sBold.Render(trunc(rootName, 30)) + " " + human(total) + sDim.Render(" ▸ ") + strings.Join(legend, "  ")
	if shown < len(kids) {
		legendLine += "  " + lipgloss.NewStyle().Foreground(cFaint).Render("■") + " " + sDim.Render(T("hdr.other"))
	}
	l2 := " " + strings.Join(segs, "")
	l3 := " " + trunc(legendLine, w-2)

	// status line
	var status string
	live := m.v.Live
	switch {
	case !live.Done:
		spin := sAccent.Render(spinFrames[m.tick%len(spinFrames)])
		est := ""
		if m.v.Snap.Cached && m.v.Snap.Progress.Files > 0 {
			p := float64(live.Files) / float64(m.v.Snap.Progress.Files) * 100
			if p > 99 {
				p = 99
			}
			est = fmt.Sprintf(" ~%.0f%%", p)
		}
		status = fmt.Sprintf("%s %s%s  %s %s · %s %s · %s · %.1fs", spin, T("st.scanning"), est,
			num(live.Files), T("st.files"), num(live.Dirs), T("st.dirs"), human(live.Bytes), live.Elapsed.Seconds())
		if m.v.Snap.Cached {
			status += sDim.Render("  · " + T("st.showing_cache") + " " + ago(m.v.Snap.Taken))
		}
	default:
		status = sOK.Render("✓") + fmt.Sprintf(" %s %s %s · %s %s %s %.1fs", T("st.scanned"), num(live.Files), T("st.files"),
			num(live.Dirs), T("st.dirs"), T("st.in"), live.Elapsed.Seconds())
	}
	right := sOK.Render("≈ "+human(m.v.Reclaimable)) + " " + sDim.Render(T("st.easy"))
	if n := live.Denied; n > 0 {
		right = sWarn.Render(fmt.Sprintf("⚠ %d %s", n, T("st.denied"))) + "  " + right
	}
	l4 := " " + status
	if gap := w - lipgloss.Width(l4) - lipgloss.Width(right) - 1; gap > 0 {
		l4 += strings.Repeat(" ", gap) + right
	} else {
		l4 = trunc(l4, w)
	}

	tabs := []string{T("tab.suspects"), T("tab.explore"), T("tab.big")}
	var tl []string
	for i, t := range tabs {
		lbl := fmt.Sprintf("%d %s", i+1, t)
		if tab(i) == m.tab {
			tl = append(tl, sTabOn.Render(lbl))
		} else {
			tl = append(tl, sTabOff.Render(lbl))
		}
	}
	sel := ""
	if n := len(m.selected); n > 0 {
		sel = sAccent.Render(fmt.Sprintf("● %d %s · %s", n, T("sel.selected"), human(m.selectionSize()))) + sDim.Render("  d "+T("sel.review"))
	}
	l5 := " " + strings.Join(tl, "   ")
	if gap := w - lipgloss.Width(l5) - lipgloss.Width(sel) - 1; gap > 0 {
		l5 += strings.Repeat(" ", gap) + sel
	}
	return []string{trunc(l1, w), l2, l3, l4, l5, sDim.Render(strings.Repeat("─", w))}
}

func ago(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return T("ago.now")
	case d < time.Hour:
		return fmt.Sprintf(T("ago.min"), int(d.Minutes()))
	case d < 48*time.Hour:
		return fmt.Sprintf(T("ago.hour"), int(d.Hours()))
	}
	return fmt.Sprintf(T("ago.day"), int(d.Hours()/24))
}

func ageText(unix int64) string {
	a := session.Age(unix)
	if a == "today" {
		return T("age.today")
	}
	return a
}

// --- main ----------------------------------------------------------------------

func (m *Model) viewMain() string {
	lines := m.header()
	h := m.listHeight()
	var list []string
	var detail []string
	switch m.tab {
	case tabSuspects:
		list, detail = m.viewSuspects(h)
	case tabExplore:
		list, detail = m.viewExplore(h)
	case tabBig:
		list, detail = m.viewBig(h)
	}
	for len(list) < h {
		list = append(list, "")
	}
	lines = append(lines, list...)
	lines = append(lines, sDim.Render(strings.Repeat("─", m.w)))
	for len(detail) < footerLines-2 {
		detail = append(detail, "")
	}
	for _, d := range detail[:footerLines-2] {
		lines = append(lines, " "+trunc(d, m.w-2))
	}
	lines = append(lines, m.footer())
	return strings.Join(lines, "\n")
}

func (m *Model) footer() string {
	if m.flash != "" {
		return " " + sWarn.Render(trunc(m.flash, m.w-2))
	}
	var keys string
	switch m.tab {
	case tabSuspects:
		keys = T("keys.suspects")
	case tabExplore:
		keys = T("keys.explore")
	case tabBig:
		keys = T("keys.big")
	}
	return " " + sDim.Render(trunc(keys+"  ·  "+T("keys.global"), m.w-2))
}

func check(on, partial bool) string {
	switch {
	case on:
		return sAccent.Render("[x]")
	case partial:
		return sAccent.Render("[-]")
	}
	return sDim.Render("[ ]")
}

// cols: sel(4) label(flex) size(9) bar(barW) age(6) safety(12)
func (m *Model) itemLine(indent int, mark, label, sub string, size int64, frac float64, age, safety string, sc lipgloss.TerminalColor, done bool) string {
	barW := 14
	if m.w < 100 {
		barW = 8
	}
	fixed := 4 + 1 + 9 + 1 + barW + 1 + 6 + 1 + 12 + 1
	lw := m.w - fixed - indent
	if lw < 10 {
		lw = 10
	}
	txt := label
	if sub != "" {
		txt += "  " + sDim.Render(sub)
	}
	sz := human(size)
	if !done {
		sz = "≥" + sz
	}
	return strings.Repeat(" ", indent) + mark + " " + pad(txt, lw) + " " + padL(sz, 9) + " " +
		bar(frac, barW, sc) + " " + padL(sDim.Render(age), 6) + " " + pad(lipgloss.NewStyle().Foreground(sc).Render(safety), 12)
}

func (m *Model) viewSuspects(h int) ([]string, []string) {
	rows := m.suspectRows()
	if len(rows) == 0 {
		return []string{"", "  " + sDim.Render(T("sus.empty"))}, nil
	}
	c := m.suspectCursor(rows)
	m.sCursor = c
	m.sOffset = scroll(c, m.sOffset, h, len(rows))
	var maxGroup int64 = 1
	for _, g := range m.v.Groups {
		if g.Total > maxGroup {
			maxGroup = g.Total
		}
	}
	var out []string
	for i := m.sOffset; i < len(rows) && i < m.sOffset+h; i++ {
		r := rows[i]
		var line string
		switch r.kind {
		case rowGroup:
			arrow := "▾"
			if !m.openG[r.g.Group] {
				arrow = "▸"
			}
			name := sBold.Render(arrow + " " + T("group."+string(r.g.Group)))
			cnt := sDim.Render(fmt.Sprintf("%d %s", len(r.g.Items), T("sus.items")))
			line = m.itemLine(0, " ", name, "", r.g.Total, float64(r.g.Total)/float64(maxGroup), "", "", cAccent, true)
			line = strings.TrimRight(line, " ") + "  " + cnt
		case rowItem, rowKid:
			it := r.it
			gt := r.g.Total
			if gt == 0 {
				gt = 1
			}
			ind := 2
			label := it.Label
			if r.kind == rowKid {
				ind = 6
			}
			if len(it.Kids) > 0 {
				arrow := "▸ "
				if m.expanded[it.Key] {
					arrow = "▾ "
				}
				label = arrow + label + sDim.Render(fmt.Sprintf(" ×%d", it.Count))
			} else if it.Count > 0 {
				label += sDim.Render(fmt.Sprintf(" (%d)", it.Count))
			}
			mark := check(m.isSelected(it), m.partlySelected(it))
			if !it.Actionable() {
				mark = sDim.Render(" ℹ ")
			}
			if it.System {
				label = "🔒 " + label
			}
			sub := it.Sub
			if r.kind == rowKid {
				sub = ""
			}
			frac := float64(it.Size()) / float64(gt)
			size := it.Size()
			if it.Unknown {
				size = 0
			}
			line = m.itemLine(ind, mark, label, sub, size, frac, ageText(it.Newest()), safetyLabel(it.Safety), safetyColor(it.Safety), it.Done())
			if it.Unknown {
				line = strings.Replace(line, "      0 B", "        ?", 1)
			}
		}
		if i == c {
			line = sCursor.Render(pad(line, m.w))
		}
		out = append(out, line)
	}
	return out, m.suspectDetail(rows[c])
}

func (m *Model) suspectDetail(r row) []string {
	if r.kind == rowGroup {
		return []string{
			sBold.Render(T("group."+string(r.g.Group))) + sDim.Render(" · "+human(r.g.Total)),
			sDim.Render(T("groupdesc." + string(r.g.Group))),
			sDim.Render(T("sus.group_hint")),
		}
	}
	it := r.it
	path := it.Path
	if path == "" && len(it.Kids) > 0 {
		path = fmt.Sprintf("%d %s", len(it.Kids), T("sus.locations"))
	}
	l1 := sBold.Render(it.Label) + sDim.Render("  "+session.Tilde(path, m.sess.Home))
	l2 := it.Explain.String()
	var l3 string
	switch {
	case !it.Actionable():
		l3 = sDim.Render(T("sus.info_only"))
	case len(it.Kids) > 0:
		l3 = sDim.Render(T("sus.agg_hint"))
	case it.Mode == catalog.ModeNative:
		l3 = sDim.Render(T("sus.runs")+" ") + sCmd.Render(strings.Join(it.Native, " "))
	case it.Mode == catalog.ModeContents:
		l3 = sDim.Render(T("sus.empties")+" ") + sCmd.Render(session.Tilde(it.Path, m.sess.Home))
	default:
		l3 = sDim.Render(T("sus.removes")+" ") + sCmd.Render("rm -rf "+session.Tilde(it.Path, m.sess.Home))
	}
	if it.Cached {
		l3 += sDim.Render("  · " + T("sus.cached"))
	}
	return []string{l1, l2, l3}
}

func (m *Model) viewExplore(h int) ([]string, []string) {
	d := m.exNode
	if d == nil {
		return nil, nil
	}
	rows := m.exploreRows()
	if m.exCursor >= len(rows) {
		m.exCursor = max(0, len(rows)-1)
	}
	crumb := sBold.Render(session.Tilde(d.Path(), m.sess.Home)) + sDim.Render(fmt.Sprintf("  %s · %s %s", human(d.Size()), num(d.Files()), T("st.files")))
	if !d.Done() {
		crumb += sDim.Render("  · " + T("ex.scanning"))
	}
	out := []string{" " + trunc(crumb, m.w-2)}
	h--
	if len(rows) == 0 {
		msg := T("ex.empty")
		if d.Has(scan.FlagDenied) {
			msg = T("flash.denied")
		}
		return append(out, "", "  "+sDim.Render(msg)), nil
	}
	m.exOffset = scroll(m.exCursor, m.exOffset, h, len(rows))
	total := d.Size()
	if total == 0 {
		total = 1
	}
	barW := 20
	if m.w < 100 {
		barW = 10
	}
	for i := m.exOffset; i < len(rows) && i < m.exOffset+h; i++ {
		r := rows[i]
		frac := float64(r.size) / float64(total)
		name := r.name
		key := "f:" + filepath.Join(d.Path(), r.name)
		extra := ""
		if r.dir != nil {
			name = sBold.Render(name + "/")
			key = "e:" + r.dir.Path()
			switch {
			case r.dir.Has(scan.FlagDenied):
				extra = sWarn.Render(" 🔒 " + T("ex.noaccess"))
			case r.dir.Has(scan.FlagMount):
				extra = sDim.Render(" (" + T("ex.mount") + ")")
			case r.dir.Tag != "":
				if det := catalog.DetectorByID(r.dir.Tag); det != nil {
					extra = sCmd.Render(" ◆ " + det.Label)
				}
			}
		}
		on := m.selected[key] != nil
		line := " " + check(on, false) + " " + padL(human(r.size), 9) + " " + bar(frac, barW, cAccent) +
			" " + padL(sDim.Render(fmt.Sprintf("%4.1f%%", frac*100)), 6) + "  " + padL(sDim.Render(ageText(r.mtime)), 5) + "  " + name + extra
		if r.dir != nil {
			line += sDim.Render(fmt.Sprintf("  %s %s", num(r.files), T("st.files")))
		}
		line = trunc(line, m.w)
		if i == m.exCursor {
			line = sCursor.Render(pad(line, m.w))
		}
		out = append(out, line)
	}
	r := rows[m.exCursor]
	p := filepath.Join(d.Path(), r.name)
	det := []string{sBold.Render(r.name) + sDim.Render("  "+session.Tilde(p, m.sess.Home))}
	if r.dir != nil && r.dir.Tag != "" {
		if dd := catalog.DetectorByID(r.dir.Tag); dd != nil {
			det = append(det, dd.Explain.String())
		}
	} else {
		det = append(det, sDim.Render(fmt.Sprintf("%s · %s %s", human(r.size), T("ex.modified"), ageText(r.mtime))))
	}
	det = append(det, sDim.Render(T("ex.hint")))
	return out, det
}

func (m *Model) viewBig(h int) ([]string, []string) {
	big := m.v.Snap.Big
	if len(big) == 0 {
		return []string{"", "  " + sDim.Render(T("big.empty"))}, nil
	}
	if m.bCursor >= len(big) {
		m.bCursor = len(big) - 1
	}
	m.bOffset = scroll(m.bCursor, m.bOffset, h, len(big))
	maxSz := big[0].Size
	var out []string
	for i := m.bOffset; i < len(big) && i < m.bOffset+h; i++ {
		f := big[i]
		on := m.selected["f:"+f.Path] != nil
		line := " " + check(on, false) + " " + padL(human(f.Size), 9) + " " + bar(float64(f.Size)/float64(maxSz), 12, cReview) +
			"  " + padL(sDim.Render(ageText(f.MTime)), 5) + "  " + sBold.Render(filepath.Base(f.Path)) + "  " +
			sDim.Render(truncLeft(session.Tilde(filepath.Dir(f.Path), m.sess.Home), max(10, m.w-60-len(filepath.Base(f.Path)))))
		line = trunc(line, m.w)
		if i == m.bCursor {
			line = sCursor.Render(pad(line, m.w))
		}
		out = append(out, line)
	}
	f := big[m.bCursor]
	return out, []string{
		sBold.Render(filepath.Base(f.Path)) + sDim.Render("  "+session.Tilde(f.Path, m.sess.Home)),
		sDim.Render(fmt.Sprintf("%s · %s %s", human(f.Size), T("ex.modified"), time.Unix(f.MTime, 0).Format("2006-01-02"))),
		sDim.Render(T("big.hint")),
	}
}

// --- dry run -------------------------------------------------------------------

func (m *Model) viewDryRun() string {
	p := m.plan
	w := m.w
	nReady, szReady := p.Count(plan.Ready)
	nRoot, szRoot := p.Count(plan.NeedsRoot)
	nBlk, _ := p.Count(plan.Blocked)
	lines := []string{
		" " + sTitle.Render(T("dr.title")) + "  " + sWarn.Render(T("dr.nothing_yet")),
		"",
		" " + sOK.Render(fmt.Sprintf("%d %s", nReady, T("dr.will_run"))) + sDim.Render(" · ") + sBold.Render("≈ "+human(szReady)) +
			sDim.Render(fmt.Sprintf("   │  %d %s (%s)   │  %d %s", nRoot, T("dr.need_sudo"), human(szRoot), nBlk, T("dr.blocked"))),
		sDim.Render(strings.Repeat("─", w)),
	}
	// one block per action: label line, the full command (wrapped), a blank line
	avail := m.h - len(lines) - 3
	blocks := make([][]string, len(p.Actions))
	for i, a := range p.Actions {
		blocks[i] = m.actionBlock(a, i == m.dCursor)
	}
	if m.dCursor < m.dOffset {
		m.dOffset = m.dCursor
	}
	for m.dOffset < m.dCursor {
		used := 0
		for i := m.dOffset; i <= m.dCursor; i++ {
			used += len(blocks[i])
		}
		if used <= avail {
			break
		}
		m.dOffset++
	}
	used := 0
	for i := m.dOffset; i < len(blocks); i++ {
		if used+len(blocks[i]) > avail && i > m.dOffset {
			break
		}
		lines = append(lines, blocks[i]...)
		used += len(blocks[i])
	}
	for len(lines) < m.h-2 {
		lines = append(lines, "")
	}
	lines = lines[:m.h-2]
	lines = append(lines, sDim.Render(strings.Repeat("─", w)))
	switch {
	case m.confirm && m.sess.ID.Root:
		lines = append(lines, " "+sBadgeRed.Render("ROOT")+" "+sDanger.Render(fmt.Sprintf(T("dr.confirm_root"), nReady, human(szReady)))+" "+m.confirmText+"▏")
	case m.confirm:
		lines = append(lines, " "+sDanger.Render(fmt.Sprintf(T("dr.confirm"), nReady, human(szReady))))
	case m.flash != "":
		lines = append(lines, " "+sWarn.Render(trunc(m.flash, w-2)))
	default:
		keys := T("keys.dryrun")
		if m.opts.DryRun {
			keys = T("keys.dryrun_only")
		}
		lines = append(lines, " "+sDim.Render(trunc(keys, w-2)))
	}
	return strings.Join(lines, "\n")
}

func (m *Model) actionBlock(a *plan.Action, cur bool) []string {
	w := m.w
	var icon string
	switch a.State {
	case plan.Ready:
		icon = sOK.Render("●")
	case plan.NeedsRoot:
		icon = sWarn.Render("🔒")
	default:
		icon = sDanger.Render("✗")
	}
	label := sBold.Render(a.Item.Label)
	if a.Item.Sub != "" {
		label += sDim.Render(" · " + a.Item.Sub)
	}
	kind := ""
	switch a.Kind {
	case plan.Trash:
		kind = sAccent.Render(" [" + T("dr.to_trash") + "]")
	case plan.Native:
		kind = sAccent.Render(" [" + T("dr.native") + "]")
	case plan.Contents:
		kind = sAccent.Render(" [" + T("dr.contents") + "]")
	}
	sz := human(a.Size)
	if a.Item.Unknown {
		sz = "?"
	}
	out := []string{trunc(" "+icon+" "+padL(sz, 9)+"  "+label+kind+"  "+
		lipgloss.NewStyle().Foreground(safetyColor(a.Item.Safety)).Render(safetyLabel(a.Item.Safety)), w)}
	switch a.State {
	case plan.Blocked:
		out = append(out, trunc("      "+sDanger.Render(T("dr.blocked_because")+" "+a.Reason), w))
		for _, l := range wrap("# "+a.Command(m.plan.Home), w-8) {
			out = append(out, "      "+sDim.Render(l))
		}
	default:
		for i, l := range wrap("$ "+a.Command(m.plan.Home), w-8) {
			if i > 0 {
				l = "  " + l
			}
			out = append(out, "      "+sCmd.Render(l))
		}
		if a.State == plan.NeedsRoot {
			out = append(out, "      "+sDim.Render("("+T("dr.sudo_note")+")"))
		}
	}
	if cur {
		for i := range out {
			out[i] = sCursor.Render(pad(out[i], w))
		}
	}
	return append(out, "")
}

// wrap hard-wraps plain text to w cells (commands must be shown in full).
func wrap(s string, w int) []string {
	if w < 10 {
		w = 10
	}
	r := []rune(s)
	var out []string
	for len(r) > w {
		out = append(out, string(r[:w]))
		r = r[w:]
	}
	return append(out, string(r))
}

// --- applying & report ---------------------------------------------------------

func resultLine(r apply.Result, home string, w int) string {
	a := r.Action
	label := a.Item.Label
	if a.Item.Sub != "" {
		label += " · " + a.Item.Sub
	}
	switch {
	case r.OK:
		return trunc(" "+sOK.Render("✓")+" "+padL(human(r.Freed), 9)+"  "+label+sDim.Render(fmt.Sprintf("  %.1fs", r.Took.Seconds())), w)
	case r.Skipped != "":
		return trunc(" "+sWarn.Render("–")+" "+padL("", 9)+"  "+label+sDim.Render("  "+T("rep.skipped")+": "+r.Skipped), w)
	}
	msg := ""
	if r.Err != nil {
		msg = r.Err.Error()
	}
	return trunc(" "+sDanger.Render("✗")+" "+padL("", 9)+"  "+label+sDanger.Render("  "+msg), w)
}

func (m *Model) viewApplying() string {
	total := len(m.plan.Actions)
	done := len(m.results)
	var freed int64
	for _, r := range m.results {
		freed += r.Freed
	}
	frac := 0.0
	if total > 0 {
		frac = float64(done) / float64(total)
	}
	lines := []string{
		" " + sTitle.Render(T("ap.title")),
		"",
		" " + bar(frac, max(10, m.w-30), cAccent) + fmt.Sprintf("  %d/%d · %s", done, total, human(freed)),
		"",
	}
	if m.applyIdx < total && done < total {
		a := m.plan.Actions[m.applyIdx]
		lines = append(lines, " "+sAccent.Render(spinFrames[m.tick%len(spinFrames)])+" "+trunc(a.Command(m.plan.Home), m.w-4), "")
	}
	start := 0
	if room := m.h - len(lines) - 1; len(m.results) > room && room > 0 {
		start = len(m.results) - room
	}
	for _, r := range m.results[start:] {
		lines = append(lines, resultLine(r, m.plan.Home, m.w))
	}
	return strings.Join(lines, "\n")
}

func (m *Model) viewReport() string {
	var freed int64
	var ok, skipped, failed int
	for _, r := range m.results {
		freed += r.Freed
		switch {
		case r.OK:
			ok++
		case r.Skipped != "":
			skipped++
		default:
			failed++
		}
	}
	lines := []string{" " + sTitle.Render(T("rep.title")), ""}
	if m.applyErr != nil {
		lines = append(lines, " "+sDanger.Render(m.applyErr.Error()), "")
	}
	lines = append(lines,
		" "+sOK.Render(fmt.Sprintf("%s ≈ %s", T("rep.freed"), human(freed)))+
			sDim.Render(fmt.Sprintf("   %d ok · %d %s · %d %s", ok, skipped, T("rep.skipped"), failed, T("rep.failed"))),
		" "+fmt.Sprintf("%s: %s %s → %s %s", T("rep.disk"), human(m.diskBefore.Avail), T("hdr.free"), sBold.Render(human(m.diskAfter.Avail)), T("hdr.free")),
		"",
	)
	for _, r := range m.results {
		lines = append(lines, resultLine(r, m.plan.Home, m.w))
		if r.Output != "" && (r.Err != nil || r.Action.Kind == plan.Native) {
			for _, o := range strings.Split(r.Output, "\n") {
				lines = append(lines, "      "+sDim.Render(trunc(o, m.w-8)))
			}
		}
	}
	lines = append(lines, "")
	if m.sudoScript != "" {
		lines = append(lines, " "+sWarn.Render("🔒 "+T("rep.sudo_saved")+" ")+sCmd.Render(m.sudoScript),
			" "+sDim.Render(T("rep.sudo_hint")+" sudo sh "+plan.Q(m.sudoScript)))
	}
	lines = append(lines, " "+sDim.Render(T("rep.log")+" "+apply.LogPath(m.sess.ID)))
	if m.sess.ID.Root || failed > 0 {
		lines = append(lines, " "+sDim.Render(T("rep.rescan_hint")))
	}
	lines = append(lines, "", " "+sDim.Render(T("keys.report")))
	return strings.Join(lines, "\n")
}

// --- help ----------------------------------------------------------------------

func (m *Model) viewHelp() string {
	lines := []string{" " + sTitle.Render("mr-plow · "+T("help.title")), ""}
	for _, l := range strings.Split(T("help.body"), "\n") {
		lines = append(lines, "  "+l)
	}
	lines = append(lines, "", "  "+sDim.Render(T("help.safety")), "", "  "+sDim.Render(T("help.close")))
	return fitHeight(strings.Join(lines, "\n"), m.h)
}
