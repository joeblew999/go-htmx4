// Command seo reports and nudges how Google indexes a site, through the Search Console API (package
// seo), and audits it live as Googlebot sees it:
//
//	fnox exec -- go tool seo sites                # properties the service account can use; exit 1 without Full access
//	fnox exec -- go tool seo submit               # submit <base>/sitemap.xml
//	fnox exec -- go tool seo status               # sitemap status + index status of every sitemap URL
//	fnox exec -- go tool seo [-open] todo         # what Google hasn't indexed; exit 1 (and -open) on problems
//	fnox exec -- go tool seo [-open] inspect /de/ # one URL: Google's verdict + its Search Console page
//	go tool seo audit                             # live fetch as Googlebot: robots.txt + every sitemap URL
//	fnox exec -- go tool seo lighthouse [/de/]    # Lighthouse on Google's servers (PageSpeed Insights): scores, failures
//	fnox exec -- go tool seo badges               # the README's badges and scores table, from Lighthouse and the audit
//	go tool seo init                              # seo.toml, the mise tasks and the agent skill, into an app
//	go tool seo tasks                             # write its mise tasks: .mise/tasks/seo.toml
//	go tool seo skill                             # install its agent skill: .claude/skills/seo
//
// Flags go before or after the command (seo todo -open). -json prints one JSON document on stdout instead of
// text (same data, for agents, CI and jq) and never opens a browser.
// Exit codes are the contract between tasks: 0 ok, 1 problems (or no access), 2 usage.
//
// The key is GOOGLE_SEARCH_CONSOLE_KEY (a service account JSON key, e.g. from fnox); audit needs none. The settings
// come from the app's seo.toml (package config), each overridden by its flag: -base defaults to its site (there is
// no other default, and every run says which site it works on), and -site to its property, else the Domain property of the site's registrable domain
// (app.example.com → sc-domain:example.com). Request indexing has no API: todo prints Google's link for each problem
// (-open opens only those).
package main

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/joeblew999/go-htmx4/seo/audit"
	"github.com/joeblew999/go-htmx4/seo/config"
	"github.com/joeblew999/go-htmx4/seo/mise"
	"github.com/joeblew999/go-htmx4/seo/searchconsole"
	"github.com/joeblew999/go-htmx4/seo/skills"
)

// JSON reports (-json). Field names are the stable interface; text output renders the same values.
type (
	auditReport struct {
		Base     string   `json:"base"`
		URLs     int      `json:"urls"`
		OK       bool     `json:"ok"`
		Problems []string `json:"problems"`
	}
	sitesReport struct {
		Site   string               `json:"site"`
		Access bool                 `json:"access"` // Full or Owner on Site
		Sites  []searchconsole.Site `json:"sites"`
	}
	submitReport struct {
		Site    string `json:"site"`
		Sitemap string `json:"sitemap"`
	}
	urlStatus struct {
		URL             string `json:"url"`
		Verdict         string `json:"verdict,omitempty"`
		Coverage        string `json:"coverage,omitempty"`
		LastCrawl       string `json:"lastCrawl,omitempty"`
		GoogleCanonical string `json:"googleCanonical,omitempty"`
		UserCanonical   string `json:"userCanonical,omitempty"`
		Link            string `json:"link,omitempty"`
		Error           string `json:"error,omitempty"`
	}
	statusReport struct {
		Sitemap      *searchconsole.Sitemap `json:"sitemap"`
		SitemapError string                 `json:"sitemapError,omitempty"`
		URLs         []urlStatus            `json:"urls"`
		Counts       map[string]int         `json:"counts"`
	}
	inspectReport struct {
		URL string `json:"url"`
		searchconsole.Inspection
		Link string `json:"link"`
	}
	todoReport struct {
		Total    int       `json:"total"`
		Indexed  int       `json:"indexed"`
		Problems int       `json:"problems"`
		Findings []finding `json:"findings"`
	}
)

func main() {
	cfg, err := config.Find(".")
	if err != nil {
		fail(err)
	}
	host := ""
	if u, err := url.Parse(cfg.Site); err == nil {
		host = u.Hostname()
	}
	site := flag.String("site", cmp.Or(cfg.Property, domainProperty(host)), "Search Console property (sc-domain:… or a URL prefix)")
	base := flag.String("base", cfg.Site, "the app's base URL (sitemap at <base>/sitemap.xml)")
	open := flag.Bool("open", false, "todo, inspect: open Google's Search Console page (todo: only for problem URLs)")
	jsonOut := flag.Bool("json", false, "print one JSON document on stdout instead of text; never opens a browser")
	out := flag.String("o", filepath.Join(".mise", "tasks", "seo.toml"), "tasks: the file to write")
	strategy := flag.String("strategy", cfg.Lighthouse.Strategy, "lighthouse: mobile (what Google indexes) or desktop")
	minScore := flag.Int("min", cfg.Lighthouse.Min, "lighthouse: the lowest score out of 100 a category may have")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: seo audit|lighthouse [path or URL]|badges|todo|inspect <path or URL>|status|sites|submit|init|tasks|skill [-site sc-domain:example.com] [-base https://app.example.com] [-open] [-json] [-o file]")
		flag.PrintDefaults()
	}
	flag.Parse()
	args := commandArgs()
	if len(args) == 1 && args[0] == "tasks" {
		writeFile(*out, mise.Tasks)
		return
	}
	if len(args) == 1 && args[0] == "skill" {
		writeSkill(filepath.Join(".claude", "skills"))
		return
	}
	if len(args) == 1 && args[0] == "init" {
		switch {
		case cfg.Path != "":
			fmt.Println("✓ " + cfg.Path + " is there: yours to edit")
		case *base != "":
			writeFile(config.File, []byte(config.TemplateFor(strings.TrimRight(*base, "/"))))
		default:
			writeFile(config.File, []byte(config.Template))
			fmt.Println("· set site in " + config.File + " (or run go tool seo init -base https://your.site): every task works on it")
		}
		writeFile(*out, mise.Tasks)
		writeSkill(filepath.Join(".claude", "skills"))
		return
	}
	if len(args) != 1 && !(len(args) == 2 && (args[0] == "inspect" || args[0] == "lighthouse")) {
		flag.Usage()
		os.Exit(2)
	}
	if *base == "" {
		fail(fmt.Errorf("no site: set site in %s, or pass -base", config.File))
	}
	// Which site, and where that came from, before anything else: a run against the wrong site must be obvious.
	from := cfg.Path
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "base" {
			from = "-base"
		}
	})
	fmt.Fprintf(os.Stderr, "seo: %s (from %s; Search Console property %s)\n", *base, from, *site)
	if *jsonOut {
		*open = false
	}
	ctx := context.Background()
	httpc := &http.Client{Timeout: 60 * time.Second}
	sitemap := strings.TrimRight(*base, "/") + "/sitemap.xml"
	if args[0] == "audit" {
		urls, problems, err := audit.Audit(ctx, httpc, *base)
		if err != nil {
			fail(err)
		}
		r := auditReport{Base: *base, URLs: urls, Problems: problems}
		r.OK = len(r.Problems) == 0
		if r.Problems == nil {
			r.Problems = []string{}
		}
		if *jsonOut {
			emit(r)
		} else if r.OK {
			fmt.Printf("✓ robots.txt and all %d sitemap URLs pass (fetched as Googlebot: 200, indexable, self-canonical, lang, "+
				"title, description, one h1, reciprocal hreflang with x-default, icon, Open Graph; a missing page is 404, "+
				"http moves to https, the home page names the site)\n", r.URLs)
		} else {
			for _, p := range r.Problems {
				fmt.Println("✗ " + p)
			}
			fmt.Printf("%d problem(s) across robots.txt and %d sitemap URLs (fetched as Googlebot)\n", len(r.Problems), r.URLs)
		}
		if !r.OK {
			os.Exit(1)
		}
		return
	}
	raw := os.Getenv("GOOGLE_SEARCH_CONSOLE_KEY")
	if raw == "" {
		fail(fmt.Errorf("GOOGLE_SEARCH_CONSOLE_KEY is not set (run through fnox exec)"))
	}
	key, err := searchconsole.ParseKey([]byte(raw))
	if err != nil {
		fail(err)
	}
	if args[0] == "lighthouse" || args[0] == "badges" {
		pages := cfg.Lighthouse.Pages
		if len(args) == 2 {
			pages = []string{args[1]}
		}
		if len(pages) == 0 {
			if pages, err = audit.SitemapPages(ctx, httpc, sitemap); err != nil {
				fail(err)
			}
		}
		for i, p := range pages {
			if strings.HasPrefix(p, "/") {
				pages[i] = strings.TrimRight(*base, "/") + p
			}
		}
		if args[0] == "badges" {
			badges(ctx, httpc, key, *base, pages, *strategy, cfg.Badges.Dir, cfg.Badges.Readme)
			return
		}
		lighthouseRun(ctx, key, pages, *strategy, *minScore, *jsonOut)
		return
	}
	token, err := key.Token(ctx, httpc, searchconsole.Scope)
	if err != nil {
		fail(err)
	}
	c := &searchconsole.Client{HTTP: httpc, Token: token, Site: *site}

	switch args[0] {
	case "sites":
		sites, err := c.Sites(ctx)
		if err != nil {
			fail(err)
		}
		r := sitesReport{Site: *site, Sites: sites}
		if r.Sites == nil {
			r.Sites = []searchconsole.Site{}
		}
		for _, s := range sites {
			if s.SiteURL == *site && (s.PermissionLevel == "siteOwner" || s.PermissionLevel == "siteFullUser") {
				r.Access = true
			}
		}
		if *jsonOut {
			emit(r)
		} else {
			for _, s := range sites {
				fmt.Printf("%s\t%s\n", s.SiteURL, s.PermissionLevel)
			}
			if !r.Access {
				fmt.Printf("✗ %s has no Full access to %s: Search Console → Settings → Users and permissions → Add user (Full)\n", key.ClientEmail, *site)
			}
		}
		if !r.Access {
			os.Exit(1)
		}
	case "submit":
		if err := c.SubmitSitemap(ctx, sitemap); err != nil {
			fail(err)
		}
		if *jsonOut {
			emit(submitReport{Site: *site, Sitemap: sitemap})
		} else {
			fmt.Printf("✓ submitted %s to %s\n", sitemap, *site)
		}
	case "status":
		status(ctx, c, httpc, sitemap, *jsonOut)
	case "inspect":
		u := args[1]
		if strings.HasPrefix(u, "/") {
			u = strings.TrimRight(*base, "/") + u
		}
		in, err := c.Inspect(ctx, u, "en")
		if err != nil {
			fail(err)
		}
		if *jsonOut {
			emit(inspectReport{URL: u, Inspection: in, Link: in.Link})
			return
		}
		fmt.Printf("%s\n  verdict    %s\n  coverage   %s\n  last crawl %s as %s (fetch %s, robots.txt %s)\n  canonical  Google: %s, yours: %s\n  %s\n",
			u, in.Verdict, in.CoverageState, or(in.LastCrawlTime, "never"), or(in.CrawledAs, "-"), or(in.PageFetchState, "-"),
			or(in.RobotsTxtState, "-"), or(in.GoogleCanonical, "-"), or(in.UserCanonical, "-"), in.Link)
		fmt.Println("  On that page: TEST LIVE URL (Google fetches the page now), then REQUEST INDEXING if it isn't indexed.")
		if *open && in.Link != "" {
			openInBrowser([]string{in.Link})
		}
	case "todo":
		urls, err := audit.SitemapURLs(ctx, httpc, sitemap)
		if err != nil {
			fail(err)
		}
		indexed, findings := todo(ctx, c, urls)
		r := todoReport{Total: len(urls), Indexed: indexed, Findings: findings}
		if r.Findings == nil {
			r.Findings = []finding{}
		}
		for _, f := range findings {
			if f.Problem {
				r.Problems++
			}
		}
		if *jsonOut {
			emit(r)
		} else {
			links := printTodo(r.Total, r.Indexed, r.Findings)
			if r.Problems > 0 {
				fmt.Printf("%d problem(s) Google reports\n", r.Problems)
				if *open {
					openInBrowser(links)
				}
			}
		}
		if r.Problems > 0 {
			os.Exit(1)
		}
	default:
		flag.Usage()
		os.Exit(2)
	}
}

// emit writes v as indented JSON on stdout.
func emit(v any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		fail(err)
	}
}

func status(ctx context.Context, c *searchconsole.Client, httpc *http.Client, sitemap string, jsonOut bool) {
	r := statusReport{Counts: map[string]int{}}
	if s, err := c.GetSitemap(ctx, sitemap); err != nil {
		r.SitemapError = err.Error()
	} else {
		r.Sitemap = &s
	}
	if !jsonOut {
		if r.Sitemap == nil {
			fmt.Printf("sitemap %s: %s\n", sitemap, r.SitemapError)
		} else {
			s := r.Sitemap
			fmt.Printf("sitemap %s: submitted %s, downloaded %s, pending %v, errors %s, warnings %s\n",
				s.Path, or(s.LastSubmitted, "-"), or(s.LastDownloaded, "never"), s.IsPending, or(s.Errors, "0"), or(s.Warnings, "0"))
			for _, ct := range s.Contents {
				fmt.Printf("  %s: %s submitted, %s indexed\n", ct.Type, ct.Submitted, or(ct.Indexed, "?"))
			}
		}
	}
	urls, err := audit.SitemapURLs(ctx, httpc, sitemap)
	if err != nil {
		fail(err)
	}
	// The URL Inspection API takes seconds per URL; run a few at once (quota: 600 per minute, 2,000 per day) and, as text,
	// print each row as soon as it and the rows before it are done.
	rows := make([]chan urlStatus, len(urls))
	sem := make(chan struct{}, 6)
	for i, u := range urls {
		rows[i] = make(chan urlStatus, 1)
		go func(u string, out chan<- urlStatus) {
			sem <- struct{}{}
			defer func() { <-sem }()
			in, err := c.Inspect(ctx, u, "en")
			if err != nil {
				out <- urlStatus{URL: u, Error: err.Error()}
				return
			}
			out <- urlStatus{URL: u, Verdict: in.Verdict, Coverage: in.CoverageState, LastCrawl: in.LastCrawlTime,
				GoogleCanonical: in.GoogleCanonical, UserCanonical: in.UserCanonical, Link: in.Link}
		}(u, rows[i])
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', tabwriter.FilterHTML)
	if !jsonOut {
		fmt.Fprintln(w, "URL\tVERDICT\tCOVERAGE\tLAST CRAWL\tCANONICAL")
	}
	problems := 0
	for _, ch := range rows {
		u := <-ch
		r.URLs = append(r.URLs, u)
		state := u.Coverage
		if u.Error != "" {
			state = "error"
		}
		r.Counts[state]++
		canonical := "ok"
		if u.Error != "" || u.GoogleCanonical != "" && u.GoogleCanonical != u.URL {
			problems++
			canonical = "Google chose " + u.GoogleCanonical
		}
		if !jsonOut {
			if u.Error != "" {
				fmt.Fprintf(w, "%s\tERROR\t%s\t\t\n", u.URL, u.Error)
			} else {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", u.URL, u.Verdict, u.Coverage, or(u.LastCrawl, "-"), canonical)
			}
			w.Flush()
		}
	}
	if jsonOut {
		emit(r)
		return
	}
	for state, n := range r.Counts {
		fmt.Printf("  %3d  %s\n", n, state)
	}
	fmt.Printf("%d URLs inspected (Google's index, not a live fetch); %d with errors or a different canonical\n", len(urls), problems)
}

// domainProperty is the Domain property of a host's registrable domain: app.example.com →
// sc-domain:example.com (last two labels; good enough for .net/.com hosts, pass -site otherwise).
func domainProperty(host string) string {
	labels := strings.Split(host, ".")
	if len(labels) < 2 {
		return ""
	}
	return "sc-domain:" + strings.Join(labels[len(labels)-2:], ".")
}

func or(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "seo:", err)
	os.Exit(1)
}

// commandArgs is the command and its argument, with flags allowed after them as well as before: seo todo
// -open is seo -open todo.
func commandArgs() []string {
	var args []string
	for rest := flag.Args(); len(rest) > 0; rest = flag.Args() {
		args = append(args, rest[0])
		flag.CommandLine.Parse(rest[1:])
	}
	return args
}

// writeSkill installs seo's agent skill into dir (an app's .claude/skills).
func writeSkill(dir string) {
	err := fs.WalkDir(skills.FS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := skills.FS.ReadFile(path)
		if err == nil {
			writeFile(filepath.Join(dir, path), b)
		}
		return err
	})
	if err != nil {
		fail(err)
	}
}

// writeFile writes b to path, unless it holds b already.
func writeFile(path string, b []byte) {
	if old, err := os.ReadFile(path); err == nil && bytes.Equal(old, b) {
		fmt.Printf("✓ %s is up to date\n", path)
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		fail(err)
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		fail(err)
	}
	fmt.Println("wrote", path)
}
