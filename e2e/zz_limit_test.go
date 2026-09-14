package e2e

import (
	"encoding/json"
	"testing"

	"github.com/chromedp/chromedp"
)

// TestWriteLimit hammers the board past the per-client write limit: 429s arrive as "Slow down" toasts and the
// board stops changing. It uses up the client's write budget, so it runs last (file name order).
func TestWriteLimit(t *testing.T) {
	b := newTab(t, "browser")
	b.allow4xx = true
	b.run(chromedp.Navigate(base+"/board?topic="+topic("e2e-limit")), chromedp.WaitVisible("#board", chromedp.ByQuery))
	var r struct {
		Codes  map[string]int
		Toasts int
		Value  int
	}
	raw := b.str(`(async () => {
		const codes = {};
		document.addEventListener("htmx:after:request", (e) => { const c = e.detail.ctx?.response?.status; codes[c] = (codes[c] ?? 0) + 1; });
		const btn = document.querySelector('button[aria-label="Increment"]');
		for (let i = 0; i < 80; i++) { btn.click(); await new Promise(r => setTimeout(r, 40)); }
		await new Promise(r => setTimeout(r, 1500));
		const toasts = [...document.querySelectorAll("#gsxui-toaster [data-gsxui-slot-toast]")].filter(t => t.textContent.includes("Slow down")).length;
		return JSON.stringify({codes, toasts, value: Number(document.querySelector("#board output").textContent)});
	})()`)
	if err := json.Unmarshal([]byte(raw), &r); err != nil {
		t.Fatalf("%v: %s", err, raw)
	}
	check(t, "some writes pass, the rest get 429", r.Codes["200"] > 0 && r.Codes["429"] > 0, raw)
	check(t, `every 429 shows a "Slow down" toast`, r.Toasts == r.Codes["429"], raw)
	check(t, "the board only counts accepted writes", r.Value == r.Codes["200"], raw)
	b.noProblems()
}
