package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/joeblew999/go-htmx4/seo/lighthouse"
	"github.com/joeblew999/go-htmx4/seo/searchconsole"
)

// lighthouseRun runs Lighthouse on pages through PageSpeed Insights as auth, four at a time, prints each page's four
// scores, what fails, and real users' Core Web Vitals, and exits 1 when a category of any page scores under min.
func lighthouseRun(ctx context.Context, auth lighthouse.Auth, pages []string, strategy string, min int, jsonOut bool) {
	reports, errs := lighthouseAll(ctx, auth, pages, strategy)
	low := false
	for i, r := range reports {
		if errs[i] != nil {
			low = true
			if !jsonOut {
				fmt.Printf("✗ %s: %v\n", pages[i], errs[i])
			}
			continue
		}
		for _, c := range r.Categories {
			low = low || c.Score < min
		}
		if !jsonOut {
			printLighthouse(r, min)
		}
	}
	if jsonOut {
		emit(reports)
	}
	if low {
		os.Exit(1)
	}
}

// lighthouseAuth is how Lighthouse runs: with PAGESPEED_API_KEY when it's set (a free key is all it needs), else with
// the Search Console service account's token (GOOGLE_SEARCH_CONSOLE_KEY).
func lighthouseAuth(ctx context.Context) lighthouse.Auth {
	if k := os.Getenv("PAGESPEED_API_KEY"); k != "" {
		return lighthouse.Auth{Key: k}
	}
	raw := os.Getenv("GOOGLE_SEARCH_CONSOLE_KEY")
	if raw == "" {
		fail(fmt.Errorf("Lighthouse needs PAGESPEED_API_KEY (a free key: Google Cloud console → APIs & Services → " +
			"Credentials → API key, restricted to the PageSpeed Insights API) or GOOGLE_SEARCH_CONSOLE_KEY, run through fnox exec"))
	}
	key, err := searchconsole.ParseKey([]byte(raw))
	if err != nil {
		fail(err)
	}
	token, err := key.Token(ctx, &http.Client{Timeout: time.Minute}, lighthouse.Scope)
	if err != nil {
		fail(err)
	}
	return lighthouse.Auth{Token: token}
}

// lighthouseAll runs Lighthouse on every page, four at a time.
func lighthouseAll(ctx context.Context, auth lighthouse.Auth, pages []string, strategy string) ([]lighthouse.Report, []error) {
	httpc := &http.Client{Timeout: 3 * time.Minute}
	reports, errs := make([]lighthouse.Report, len(pages)), make([]error, len(pages))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 4)
	for i, page := range pages {
		wg.Go(func() {
			sem <- struct{}{}
			defer func() { <-sem }()
			reports[i], errs[i] = lighthouse.Run(ctx, httpc, auth, page, strategy)
		})
	}
	wg.Wait()
	return reports, errs
}

func printLighthouse(r lighthouse.Report, min int) {
	fmt.Printf("%s (%s, Lighthouse on Google's servers)\n", r.URL, r.Strategy)
	for _, c := range r.Categories {
		mark := "✓"
		if c.Score < min {
			mark = "✗"
		}
		fmt.Printf("%s %-15s %3d\n", mark, c.Title, c.Score)
		for _, a := range c.Failed {
			fmt.Printf("      %3d  %s  %s  (%s)\n", a.Score, a.Title, a.Display, a.ID)
			for _, it := range a.Items {
				fmt.Printf("             %.160s\n", it)
			}
		}
	}
	if len(r.Field) == 0 {
		fmt.Println("real users: too few visits for Chrome's field data yet")
	}
	for _, m := range r.Field {
		fmt.Printf("real users: %-40s p75 %6d  %s\n", m.ID, m.Percentile, m.Category)
	}
	fmt.Println()
}
