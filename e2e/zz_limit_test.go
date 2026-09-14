package e2e

import (
	"encoding/json"
	"testing"

	"github.com/chromedp/chromedp"
)

// TestWriteLimit uses up this client's write budget with parallel writes, then clicks Increment through htmx
// once the limiter answers 429: the click must get 429, show a "Slow down" toast and leave the board alone.
// Parallel fetches, not clicks: htmx won't start a request from a button while its previous one is in flight,
// so clicks can't outrun a real network. It uses up the budget, so it runs last (file name order).
func TestWriteLimit(t *testing.T) {
	b := newTab(t, "browser")
	b.allow4xx = true
	top := topic("e2e-limit")
	b.run(chromedp.Navigate(base+"/board?topic="+top), chromedp.WaitVisible("#board", chromedp.ByQuery))
	var r struct {
		Burst       map[string]int
		ClickStatus int
		Attempts    int
		Toasts      int
		Before      string // board version in D1 before the click
		After       string
	}
	raw := b.str(`(async () => {
		const url = "/board/add?topic=` + top + `";
		const post = () => fetch(url, {method: "POST", headers: {"Content-Type": "application/x-www-form-urlencoded"}, body: "delta=1"}).then(res => res.status);
		const version = async () => (await (await fetch("/board?topic=` + top + `")).text()).match(/data-version="(\d+)"/)[1];
		const burst = {};
		const tally = (s) => { burst[s] = (burst[s] ?? 0) + 1; return s; };
		// Cloudflare's limiter is per location and eventually consistent, with fixed windows: send parallel batches
		// (one at a time would stay under 60 per 10 s over a slow network) until one answers 429, then click at once;
		// retry if a window boundary gets in between.
		let clickStatus = 0, before = "", after = "", attempts = 0;
		while (clickStatus !== 429 && attempts++ < 5) {
			for (let batch = 0; batch < 10; batch++) {
				const got = await Promise.all(Array.from({length: 20}, post));
				got.forEach(tally);
				if (got.includes(429)) break;
			}
			before = await version();
			const done = new Promise(res => document.addEventListener("htmx:after:request", (e) => res(e.detail.ctx?.response?.status), {once: true}));
			document.querySelector('button[aria-label="Increment"]').click();
			clickStatus = await done;
			after = await version();
		}
		await new Promise(r => setTimeout(r, 800));
		const toasts = [...document.querySelectorAll("#gsxui-toaster [data-gsxui-slot-toast]")].filter(t => t.textContent.includes("Slow down")).length;
		return JSON.stringify({burst, clickStatus, attempts, toasts, before, after});
	})()`)
	if err := json.Unmarshal([]byte(raw), &r); err != nil {
		t.Fatalf("%v: %s", err, raw)
	}
	check(t, "a burst gets some writes through and the rest 429", r.Burst["200"] > 0 && r.Burst["429"] > 0, raw)
	check(t, "the htmx click over the limit gets 429", r.ClickStatus == 429, raw)
	check(t, `and shows a "Slow down" toast`, r.Toasts >= 1, raw)
	check(t, "and leaves the board unchanged (D1 version)", r.After == r.Before, raw)
	b.noProblems()
}
