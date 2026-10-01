// mr-plow finds what is eating your disk and lets you remove it, with a dry
// run of the exact commands first.
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/ivaneidel/mr-plow/internal/catalog"
	"github.com/ivaneidel/mr-plow/internal/i18n"
	"github.com/ivaneidel/mr-plow/internal/plan"
	"github.com/ivaneidel/mr-plow/internal/safety"
	"github.com/ivaneidel/mr-plow/internal/scan"
	"github.com/ivaneidel/mr-plow/internal/session"
	"github.com/ivaneidel/mr-plow/internal/ui"
)

var version = "dev"

func main() {
	var (
		root      = flag.String("root", "", "folder to scan (default: your home)")
		lang      = flag.String("lang", "", "UI language: en or es (default: from $LANG)")
		dryRun    = flag.Bool("dry-run", false, "never apply anything; only show and export the plan")
		noCache   = flag.Bool("no-cache", false, "don't load or save the last-scan cache")
		bench     = flag.Bool("bench", false, "benchmark the scanner against du on --root and exit")
		report    = flag.Bool("report", false, "scan, print the suspects as text and exit (no UI)")
		scriptOut = flag.String("script-out", "mr-plow-plan.sh", "where the dry-run script is exported")
		lister    = flag.String("lister", "", "force a directory lister (debug): "+listerNames())
		showVer   = flag.Bool("version", false, "print version and exit")
	)
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "mr-plow %s: find what eats your disk, see the exact rm, then decide.\n\nUsage: mr-plow [flags]\n\n", version)
		flag.PrintDefaults()
	}
	flag.Parse()
	if *showVer {
		fmt.Println("mr-plow", version)
		return
	}
	if os.Getenv("MR_PLOW_DRY_RUN") == "1" {
		*dryRun = true
	}
	i18n.Set(*lang)
	id := safety.Identify()

	if *bench {
		r := *root
		if r == "" {
			r = id.Home
		}
		if err := scan.Bench(os.Stdout, r, 2); err != nil {
			fail(err)
		}
		return
	}

	var l scan.Lister
	if *lister != "" {
		if l = scan.ListerByName(*lister); l == nil {
			fail(fmt.Errorf("unknown lister %q (have: %s)", *lister, listerNames()))
		}
	}
	sess, err := session.New(session.Options{Root: *root, NoCache: *noCache || *report, Lister: l, Identity: id})
	if err != nil {
		fail(err)
	}
	if err := sess.Start(); err != nil {
		fail(err)
	}
	if *report {
		printReport(sess)
		return
	}
	if err := ui.Run(sess, ui.Options{DryRun: *dryRun, ScriptOut: *scriptOut}); err != nil {
		fail(err)
	}
}

func listerNames() string {
	var n []string
	for _, l := range scan.Listers() {
		n = append(n, l.Name())
	}
	return strings.Join(n, ", ")
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "mr-plow:", err)
	os.Exit(1)
}

func printReport(sess *session.Session) {
	sess.WaitAll()
	v := sess.View()
	p := v.Live
	fmt.Printf("mr-plow · %s · %s files, %s folders in %.1fs (%s) · %d unreadable\n",
		sess.Root, fmtInt(p.Files), fmtInt(p.Dirs), p.Elapsed.Seconds(), v.Snap.Lister, p.Denied)
	fmt.Printf("disk %s: %s used, %s free of %s · ≈ %s easy to reclaim\n\n",
		v.Disk.Mount, plan.Human(v.Disk.Used()), plan.Human(v.Disk.Avail), plan.Human(v.Disk.Total), plan.Human(v.Reclaimable))
	for _, g := range v.Groups {
		fmt.Printf("%-28s %10s\n", i18n.T("group."+string(g.Group)), plan.Human(g.Total))
		for i, it := range g.Items {
			if i >= 12 {
				fmt.Printf("    … %d more\n", len(g.Items)-i)
				break
			}
			label := it.Label
			if len(it.Kids) > 0 {
				label = fmt.Sprintf("%s ×%d", label, len(it.Kids))
			}
			if it.Sub != "" {
				label += " · " + it.Sub
			}
			size := plan.Human(it.Size())
			if it.Unknown {
				size = "?"
			}
			fmt.Printf("    %10s  %-12s %-6s %s\n", size, safetyName(it.Safety), session.Age(it.Newest()), label)
		}
	}
}

func safetyName(s catalog.Safety) string {
	return [...]string{"safe", "regenerable", "review", "personal"}[s]
}

func fmtInt(n int64) string {
	s := fmt.Sprint(n)
	var b strings.Builder
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(c)
	}
	return b.String()
}
