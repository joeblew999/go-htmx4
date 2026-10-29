package e2e

import (
	"testing"
	"time"

	"github.com/chromedp/chromedp"
)

// testLiveVideo is the live page end to end (#156), through the real relay (Cloudflare's MoQ network, as the Worker
// names it): one Chrome broadcasts its built-in test camera, a second watches, and frames must decode in the watcher.
// Chrome's fake camera is its own capture path with a synthetic picture, not a stand-in for MoQ: everything from the
// encoder to the relay and back is the real thing.
func testLiveVideo(t *testing.T) {
	camera := []chromedp.ExecAllocatorOption{
		chromedp.Flag("use-fake-device-for-media-stream", true),
		chromedp.Flag("use-fake-ui-for-media-stream", true),
		chromedp.Flag("autoplay-policy", "no-user-gesture-required"),
		// MoQ runs over WebTransport, which is QUIC: newTab turns QUIC off for the other checks.
		chromedp.Flag("disable-quic", false),
	}
	caster := newTabWith(t, "broadcaster", camera...)
	caster.open("/live")
	check(t, "the broadcast starts idle", caster.waitJS(10*time.Second, `document.querySelector("[data-moq=broadcast]").dataset.state === "idle"`))
	caster.clickText("[data-moq=broadcast] button", caster.str(`document.querySelector("[data-moq=broadcast] [data-moq-action=start]").textContent.trim()`))
	onAir := caster.waitJS(60*time.Second, `document.querySelector("[data-moq=broadcast]").dataset.state === "live"`)
	check(t, "the broadcaster goes on air", onAir, caster.str(`document.querySelector("[data-moq=broadcast]").dataset.state + " " + document.querySelector("[data-moq=broadcast]").dataset.error`))

	viewer := newTabWith(t, "viewer", camera...)
	viewer.open("/live")
	playing := viewer.waitJS(60*time.Second, `document.querySelector("[data-moq=watch]").dataset.state === "playing"`)
	check(t, "the viewer plays the broadcast", playing, viewer.str(`document.querySelector("[data-moq=watch]").dataset.state`))
	frames := viewer.num(`document.querySelector("moq-watch").video.out.stats.peek()?.frameCount ?? 0`)
	check(t, "frames decode in the viewer", frames > 0, frames)

	caster.clickText("[data-moq=broadcast] button", caster.str(`document.querySelector("[data-moq=broadcast] [data-moq-action=stop]").textContent.trim()`))
	check(t, "stopping turns the camera off", caster.waitJS(10*time.Second, `document.querySelector("[data-moq=broadcast]").dataset.state === "idle"`))
	caster.noProblems()
	viewer.noProblems()
}
