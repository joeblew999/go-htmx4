package main

import (
	"context"
	"fmt"
	"net/url"
	"os/exec"
	"runtime"
	"strings"

	"github.com/joeblew999/go-htmx4/kit/searchconsole"
)

// action is one URL Google hasn't indexed as its own page, with what to do about it.
type action struct {
	url, state, what, link string
}

// todo inspects every sitemap URL in Google's index and returns the ones that need a person in Search Console (Request
// indexing has no API), plus Rich Results Test links for the live-fetch pages (Google fetching the page from its own
// machines, which is where a Cloudflare block would show).
func todo(ctx context.Context, c *searchconsole.Client, urls, live []string) (actions []action, liveLinks []string) {
	results := make([]searchconsole.Inspection, len(urls))
	errs := make([]error, len(urls))
	sem := make(chan struct{}, 6)
	done := make(chan struct{})
	for i, u := range urls {
		go func() {
			sem <- struct{}{}
			defer func() { <-sem; done <- struct{}{} }()
			results[i], errs[i] = c.Inspect(ctx, u, "en")
		}()
	}
	for range urls {
		<-done
	}
	for i, u := range urls {
		if errs[i] != nil {
			actions = append(actions, action{u, "inspection failed", errs[i].Error(), ""})
			continue
		}
		if a, ok := actionFor(c.Site, u, results[i]); ok {
			actions = append(actions, a)
		}
	}
	for _, u := range live {
		liveLinks = append(liveLinks, "https://search.google.com/test/rich-results?url="+url.QueryEscape(u))
	}
	return actions, liveLinks
}

// actionFor says what to do about a URL that isn't indexed as its own canonical page (ok=false: nothing to do).
func actionFor(site, u string, in searchconsole.Inspection) (action, bool) {
	link := in.Link
	if in.GoogleCanonical != "" && in.GoogleCanonical != u {
		return action{u, in.CoverageState, "Google indexes " + in.GoogleCanonical + " instead: the content is (nearly) the same. " +
			"Make this page's content differ (e.g. a regional copy with regional text), or accept it; hreflang still points " +
			"users there", link}, true
	}
	switch in.CoverageState {
	case "Submitted and indexed", "Indexed, not submitted in sitemap":
		return action{}, false
	case "Discovered - currently not indexed", "URL is unknown to Google", "Crawled - currently not indexed":
		return action{u, in.CoverageState, "open the link → Request indexing", link}, true
	default:
		return action{u, in.CoverageState, "open the link → check the reason, fix, then Request indexing", link}, true
	}
}

// openInBrowser opens links on macOS (open) or Linux (xdg-open); elsewhere the printed links are enough.
func openInBrowser(links []string) {
	cmd := map[string]string{"darwin": "open", "linux": "xdg-open"}[runtime.GOOS]
	if cmd == "" {
		return
	}
	for _, l := range links {
		_ = exec.Command(cmd, l).Run()
	}
}

func printTodo(actions []action, liveLinks []string) {
	if len(actions) == 0 {
		fmt.Println("✓ every sitemap URL is indexed as its own canonical page")
	}
	for _, a := range actions {
		link := a.link
		if link == "" {
			link = "(no link from Google: Search Console → URL inspection → paste the URL)"
		}
		fmt.Printf("✗ %s\n    %s\n    %s\n    %s\n", a.url, a.state, a.what, link)
	}
	fmt.Println("Live fetch by Google (Rich Results Test: the rendered HTML must be the page, not a challenge):")
	for _, l := range liveLinks {
		fmt.Println("    " + l)
	}
	if len(actions) > 0 {
		fmt.Printf("%d URL(s) need Search Console (no API for Request indexing)\n", len(actions))
	}
}

// livePages turns "/,/board,/de/" into absolute URLs on base.
func livePages(base, paths string) []string {
	var out []string
	for p := range strings.SplitSeq(paths, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, strings.TrimRight(base, "/")+p)
		}
	}
	return out
}
