// Command searchconsole reports and nudges how Google indexes the app, through the Search Console API (kit/searchconsole).
//
//	fnox exec -- go run ./cmd/searchconsole sites               # properties the service account can use
//	fnox exec -- go run ./cmd/searchconsole submit              # submit <base>/sitemap.xml
//	fnox exec -- go run ./cmd/searchconsole status              # sitemap status + index status of every sitemap URL
//
// The key is GOOGLE_SEARCH_CONSOLE_KEY (a service account JSON key, from fnox). The property defaults to the Domain
// property of APP_DOMAIN's registrable domain (sc-domain:ubuntusoftware.net) and the base URL to https://$APP_DOMAIN.
// Test live URL and Request indexing have no API: use the Search Console UI for those.
package main

import (
	"context"
	"encoding/xml"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/joeblew999/go-htmx4/kit/searchconsole"
)

func main() {
	domain := os.Getenv("APP_DOMAIN")
	site := flag.String("site", domainProperty(domain), "Search Console property (sc-domain:… or a URL prefix)")
	base := flag.String("base", "https://"+domain, "the app's base URL (sitemap at <base>/sitemap.xml)")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: searchconsole [-site sc-domain:example.com] [-base https://app.example.com] sites|submit|status")
		flag.PrintDefaults()
	}
	flag.Parse()
	if flag.NArg() != 1 || domain == "" && (*site == "" || *base == "https://") {
		flag.Usage()
		os.Exit(2)
	}
	raw := os.Getenv("GOOGLE_SEARCH_CONSOLE_KEY")
	if raw == "" {
		fail(fmt.Errorf("GOOGLE_SEARCH_CONSOLE_KEY is not set (run through fnox exec)"))
	}
	key, err := searchconsole.ParseKey([]byte(raw))
	if err != nil {
		fail(err)
	}
	ctx := context.Background()
	httpc := &http.Client{Timeout: 60 * time.Second}
	token, err := key.Token(ctx, httpc, searchconsole.Scope)
	if err != nil {
		fail(err)
	}
	c := &searchconsole.Client{HTTP: httpc, Token: token, Site: *site}
	sitemap := strings.TrimRight(*base, "/") + "/sitemap.xml"

	switch flag.Arg(0) {
	case "sites":
		sites, err := c.Sites(ctx)
		if err != nil {
			fail(err)
		}
		if len(sites) == 0 {
			fmt.Printf("%s has no properties: add it in Search Console → Settings → Users and permissions (Full)\n", key.ClientEmail)
		}
		for _, s := range sites {
			fmt.Printf("%s\t%s\n", s.SiteURL, s.PermissionLevel)
		}
	case "submit":
		if err := c.SubmitSitemap(ctx, sitemap); err != nil {
			fail(err)
		}
		fmt.Printf("✓ submitted %s to %s\n", sitemap, *site)
	case "status":
		status(ctx, c, httpc, sitemap)
	default:
		flag.Usage()
		os.Exit(2)
	}
}

func status(ctx context.Context, c *searchconsole.Client, httpc *http.Client, sitemap string) {
	if s, err := c.GetSitemap(ctx, sitemap); err != nil {
		fmt.Printf("sitemap %s: %v\n", sitemap, err)
	} else {
		fmt.Printf("sitemap %s: submitted %s, downloaded %s, pending %v, errors %s, warnings %s\n",
			s.Path, or(s.LastSubmitted, "-"), or(s.LastDownloaded, "never"), s.IsPending, or(s.Errors, "0"), or(s.Warnings, "0"))
		for _, ct := range s.Contents {
			fmt.Printf("  %s: %s submitted, %s indexed\n", ct.Type, ct.Submitted, or(ct.Indexed, "?"))
		}
	}
	urls, err := sitemapURLs(ctx, httpc, sitemap)
	if err != nil {
		fail(err)
	}
	// The URL Inspection API takes seconds per URL; run a few at once (quota: 600 per minute, 2,000 per day) and print
	// each row as soon as it and the rows before it are done.
	type row struct {
		line    string
		problem bool
	}
	rows := make([]chan row, len(urls))
	sem := make(chan struct{}, 6)
	for i, u := range urls {
		rows[i] = make(chan row, 1)
		go func(u string, out chan<- row) {
			sem <- struct{}{}
			defer func() { <-sem }()
			in, err := c.Inspect(ctx, u, "en")
			if err != nil {
				out <- row{fmt.Sprintf("%s\tERROR\t%v\t\t", u, err), true}
				return
			}
			canonical, problem := "ok", false
			if in.GoogleCanonical != "" && in.GoogleCanonical != u {
				canonical, problem = "Google chose "+in.GoogleCanonical, true
			}
			out <- row{fmt.Sprintf("%s\t%s\t%s\t%s\t%s", u, in.Verdict, in.CoverageState, or(in.LastCrawlTime, "-"), canonical), problem}
		}(u, rows[i])
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', tabwriter.FilterHTML)
	fmt.Fprintln(w, "URL\tVERDICT\tCOVERAGE\tLAST CRAWL\tCANONICAL")
	problems := 0
	counts := map[string]int{}
	for _, ch := range rows {
		r := <-ch
		fmt.Fprintln(w, r.line)
		w.Flush()
		if r.problem {
			problems++
		}
		counts[strings.Split(r.line, "\t")[2]]++
	}
	for state, n := range counts {
		fmt.Printf("  %3d  %s\n", n, state)
	}
	fmt.Printf("%d URLs inspected (Google's index, not a live fetch); %d with errors or a different canonical\n", len(urls), problems)
}

// sitemapURLs fetches the live sitemap and returns its <loc> URLs.
func sitemapURLs(ctx context.Context, httpc *http.Client, sitemap string) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", sitemap, nil)
	if err != nil {
		return nil, err
	}
	res, err := httpc.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("GET %s: %s", sitemap, res.Status)
	}
	b, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	var set struct {
		URLs []struct {
			Loc string `xml:"loc"`
		} `xml:"url"`
	}
	if err := xml.Unmarshal(b, &set); err != nil {
		return nil, fmt.Errorf("%s: %w", sitemap, err)
	}
	out := make([]string, 0, len(set.URLs))
	for _, u := range set.URLs {
		out = append(out, u.Loc)
	}
	return out, nil
}

// domainProperty is the Domain property of a host's registrable domain: go-htmx4.ubuntusoftware.net →
// sc-domain:ubuntusoftware.net (last two labels; good enough for .net/.com hosts, pass -site otherwise).
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
	fmt.Fprintln(os.Stderr, "searchconsole:", err)
	os.Exit(1)
}
