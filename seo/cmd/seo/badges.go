package main

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/joeblew999/go-htmx4/seo/audit"
	"github.com/joeblew999/go-htmx4/badge"
	"github.com/joeblew999/go-htmx4/seo/lighthouse"
	"github.com/joeblew999/go-htmx4/seo/searchconsole"
)

// The README lines between these two are seo badges' to write.
const scoresStart, scoresEnd = "<!-- seo:scores -->", "<!-- /seo:scores -->"

// categories are Lighthouse's, in its order, with the names the badges and the table use.
var categories = []struct{ id, name, title string }{
	{"performance", "performance", "Performance"}, {"accessibility", "accessibility", "Accessibility"},
	{"best-practices", "best practices", "Best practices"}, {"seo", "seo", "SEO"},
}

// badges runs Lighthouse on every page and the audit, writes an SVG badge per Lighthouse category into dir (the
// lowest page's score: a badge never flatters) and one for the audit, and rewrites readme's scores block with every
// page's scores.
func badges(ctx context.Context, httpc *http.Client, key searchconsole.Key, base string, pages []string, strategy, dir, readme string) {
	reports, errs := lighthouseAll(ctx, key, pages, strategy)
	for i, err := range errs {
		if err != nil {
			fail(fmt.Errorf("%s: %w", pages[i], err))
		}
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		fail(err)
	}
	for _, c := range categories {
		low := 100
		for _, r := range reports {
			low = min(low, score(r, c.id))
		}
		writeFile(filepath.Join(dir, "lighthouse-"+strings.ReplaceAll(c.name, " ", "-")+".svg"),
			[]byte(badge.SVG(c.name, fmt.Sprint(low), badge.ForScore(low))))
	}
	_, problems, err := audit.Audit(ctx, httpc, base)
	if err != nil {
		fail(err)
	}
	value, color := "passing", badge.Good
	if len(problems) > 0 {
		value, color = fmt.Sprintf("%d problems", len(problems)), badge.Poor
	}
	writeFile(filepath.Join(dir, "seo-audit.svg"), []byte(badge.SVG("seo audit", value, color)))
	stampScores(readme, reports, strategy)
}

// score is a report's score for category id, 0 without one.
func score(r lighthouse.Report, id string) int {
	for _, c := range r.Categories {
		if c.ID == id {
			return c.Score
		}
	}
	return 0
}

// stampScores rewrites what is between readme's scores markers: a table of every page's scores, and when and how
// they were measured. Without the markers it says where to put them.
func stampScores(readme string, reports []lighthouse.Report, strategy string) {
	b, err := os.ReadFile(readme)
	if err != nil {
		fail(err)
	}
	s := string(b)
	i, j := strings.Index(s, scoresStart), strings.Index(s, scoresEnd)
	if i < 0 || j < i {
		fmt.Printf("· %s has no %s … %s lines: put them where the scores table belongs\n", readme, scoresStart, scoresEnd)
		return
	}
	var t strings.Builder
	t.WriteString(scoresStart + "\n\n| Page |")
	for _, c := range categories {
		t.WriteString(" " + c.title + " |")
	}
	t.WriteString("\n|---|" + strings.Repeat("---:|", len(categories)) + "\n")
	for _, r := range reports {
		path := r.URL
		if u, err := url.Parse(r.URL); err == nil {
			path = u.Path
		}
		fmt.Fprintf(&t, "| [%s](%s) |", path, r.URL)
		for _, c := range categories {
			fmt.Fprintf(&t, " %d |", score(r, c.id))
		}
		t.WriteString("\n")
	}
	fmt.Fprintf(&t, "\nLighthouse %s on Google's servers (PageSpeed Insights), %s: `mise run seo:badges` writes this table and the badges.\n\n",
		strategy, time.Now().UTC().Format("2006-01-02"))
	writeFile(readme, []byte(s[:i]+t.String()+s[j:]))
}
