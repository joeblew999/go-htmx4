package e2e

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/chromedp/chromedp"
)

// TestBoardTwoBrowsers: D1 write in one browser → Room push to the other; presence, escaping, version guard.
func TestBoardTwoBrowsers(t *testing.T) {
	top := topic("e2e-board")
	a, b := newTab(t, "A"), newTab(t, "B")
	for _, tb := range []*tab{a, b} {
		tb.run(chromedp.Navigate(base+"/board?topic="+top), chromedp.WaitVisible("#board", chromedp.ByQuery))
	}
	check(t, "both opened a WebSocket", waitFor(5*time.Second, func() bool {
		a.mu.Lock()
		b.mu.Lock()
		defer a.mu.Unlock()
		defer b.mu.Unlock()
		return a.wsOpened > 0 && b.wsOpened > 0
	}))
	check(t, "presence 2 online in both", waitFor(8*time.Second, func() bool { return a.presence() == "2 online" && b.presence() == "2 online" }),
		fmt.Sprintf("(A %q, B %q)", a.presence(), b.presence()))
	check(t, "hx-ws reconnect config from the htmx-config meta",
		b.is(`htmx.config.ws?.reconnectDelay === "2s" && htmx.config.ws?.reconnectJitter === 0.5`), b.str(`JSON.stringify(htmx.config.ws ?? null)`))

	v0 := b.version()
	clicked := time.Now()
	a.run(chromedp.Click(`button[aria-label="Increment"]`, chromedp.ByQuery))
	check(t, "B updated by push after +1 in A", waitFor(5*time.Second, func() bool { return b.version() > v0 }),
		fmt.Sprintf("(v%d → v%d in %v)", v0, b.version(), time.Since(clicked).Round(time.Millisecond)))
	check(t, "A shows the same version", waitFor(3*time.Second, func() bool { return a.version() == b.version() }))

	payload := `<img src=x onerror="document.title='pwned'"> hi from B`
	b.run(chromedp.SendKeys(`input[name="body"]`, payload, chromedp.ByQuery), chromedp.Click(`form[hx-post^="/board/note"] button`, chromedp.ByQuery))
	gotNote := waitFor(5*time.Second, func() bool {
		return strings.Contains(a.str(`document.querySelector("#board [data-gsxui-slot-item]")?.textContent ?? ""`), "hi from B")
	})
	check(t, "note pushed to A as escaped text", gotNote && a.num(`document.querySelectorAll("#board img").length`) == 0 && a.str(`document.title`) != "pwned")
	check(t, "note form reset after post", waitFor(3*time.Second, func() bool { return b.str(`document.querySelector('input[name="body"]').value`) == "" }))

	// Version guard: a stale fragment must not replace the board.
	before := a.version()
	guard := a.str(`(async () => {
		const stale = '<section id="board" hx-swap-oob="true" data-version="1"><output>STALE</output></section>';
		const ev = new CustomEvent("htmx:before:swap", {cancelable: true, detail: {ctx: {text: stale}, tasks: []}});
		document.dispatchEvent(ev);
		try { await htmx.swap({text: stale, sourceElement: document.body, target: document.body, swap: "none"}); } catch (e) {}
		return JSON.stringify({prevented: ev.defaultPrevented});
	})()`)
	time.Sleep(300 * time.Millisecond)
	check(t, "stale fragment rejected by the version guard",
		strings.Contains(guard, `"prevented":true`) && a.str(`document.querySelector("#board output").textContent`) != "STALE" && a.version() == before)

	b.run(chromedp.Navigate("about:blank"))
	check(t, "presence drops to 1 when B leaves", waitFor(8*time.Second, func() bool { return a.presence() == "1 online" }), fmt.Sprintf("(A %q)", a.presence()))
	a.noProblems()
	b.noProblems()
}

// TestBoostBoard: boosted navigation into /board opens the socket, away closes it (presence), Back reconnects.
func TestBoostBoard(t *testing.T) {
	a, b := newTab(t, "A"), newTab(t, "B")
	a.run(chromedp.Navigate(base+"/board"), chromedp.WaitVisible("#board", chromedp.ByQuery))
	waitFor(3*time.Second, func() bool { var k int; _, err := fmt.Sscanf(a.presence(), "%d", &k); return err == nil })
	time.Sleep(time.Second)
	var n int
	fmt.Sscanf(a.presence(), "%d", &n) // the lobby may have other visitors
	one, two := fmt.Sprintf("%d online", n), fmt.Sprintf("%d online", n+1)

	b.open("/")
	b.eval(`window.__noReload = 1`, nil)
	nav := func(href string) {
		b.eval(fmt.Sprintf(`document.querySelector('nav a[href=%q]').click()`, href), nil)
		b.waitJS(5*time.Second, fmt.Sprintf(`location.pathname === %q`, href))
	}

	nav("/board")
	ok := b.waitJS(5*time.Second, `!!document.getElementById("board")`)
	check(t, "boosted nav to /board renders the board (no reload)", ok && b.num(`window.__noReload ?? 0`) == 1)
	check(t, "presence +1 in both", waitFor(8*time.Second, func() bool { return a.presence() == two && b.presence() == two }),
		fmt.Sprintf("(A %q, B %q)", a.presence(), b.presence()))
	v := b.version()
	a.eval(`document.querySelector('button[aria-label="Increment"]').click()`, nil)
	check(t, "push reaches the boosted page", waitFor(5*time.Second, func() bool { return b.version() > v }))

	nav("/about")
	check(t, "boosted nav away closes the socket (presence back)", waitFor(8*time.Second, func() bool { return a.presence() == one }), fmt.Sprintf("(A %q)", a.presence()))

	b.eval(`history.back()`, nil)
	b.waitJS(8*time.Second, `!!document.getElementById("board")`)
	check(t, "Back to /board reconnects", waitFor(8*time.Second, func() bool { return a.presence() == two && b.presence() == two }),
		fmt.Sprintf("(A %q, B %q)", a.presence(), b.presence()))
	v = b.version()
	a.eval(`document.querySelector('button[aria-label="Increment"]').click()`, nil)
	check(t, "push reaches B after Back", waitFor(5*time.Second, func() bool { return b.version() > v }))

	nav("/")
	check(t, "presence back when B goes home", waitFor(8*time.Second, func() bool { return a.presence() == one }), fmt.Sprintf("(A %q)", a.presence()))
	a.noProblems()
	b.noProblems()
}

// TestBareClosePresence: a browser socket closed without a status code (1005) must complete its close and
// drop out of presence (worker/room.mjs answers 1005 with 1000).
func TestBareClosePresence(t *testing.T) {
	b := newTab(t, "browser")
	b.open("/about")
	res := b.str(fmt.Sprintf(`(async () => {
		const url = location.origin.replace(/^http/, "ws") + "/live/%s";
		const open = () => new Promise((res, rej) => { const ws = new WebSocket(url); const msgs = []; ws.onmessage = (e) => msgs.push(e.data); ws.onopen = () => res({ws, msgs}); ws.onerror = rej; });
		const watcher = await open();
		const leaver = await open();
		await new Promise(r => setTimeout(r, 800));
		const closed = new Promise((res) => { leaver.ws.onclose = (e) => res("closed " + e.code); });
		leaver.ws.close();
		const result = await Promise.race([closed, new Promise(res => setTimeout(() => res("no close event"), 5000))]);
		await new Promise(r => setTimeout(r, 800));
		const presence = watcher.msgs.filter(m => m.includes("presence")).pop() ?? "";
		watcher.ws.close(1000);
		return result + " | " + presence;
	})()`, topic("e2e-close")))
	check(t, "bare close() completes and presence drops", strings.HasPrefix(res, "closed 1000") && strings.Contains(res, ">1</span>"), res)
}
