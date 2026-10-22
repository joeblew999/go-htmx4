package views

import (
	"github.com/joeblew999/go-htmx4/locales"
	"github.com/joeblew999/go-htmx4/ui"
	"github.com/joeblew999/go-htmx4/ui/icon"
)

// ThemeScript goes in <head>. Both scripts are gsxui's showcase site, copied: the paint-blocking
// theme init from site/pages/document.gsx and the toggle click handler from web/site.js, with its sync of
// same-origin frames marked data-site-isolated-preview (the /components sidebar), on a toggle and when one loads;
// gsxui e23aef1, the pin in mise.toml. The handler is delegated from <head>, so it registers once even when htmx
// swaps the body.
component ThemeScript() {
	<script>
		// Theme init — runs before first paint (blocking head script) so
		// a stored dark preference never flashes light. Explicit choice
		// (localStorage) wins; otherwise follow the OS preference.
		try {
			var gsxuiTheme = localStorage.getItem("gsxui-theme");
			if (gsxuiTheme === "dark" || (!gsxuiTheme && matchMedia("(prefers-color-scheme: dark)").matches)) {
				document.documentElement.classList.add("dark");
			}
		} catch (e) {}

		// Header theme toggle (shadcn's site model: one click flips the resolved
		// theme, no system/menu step).
		const syncIsolatedPreviewTheme = (frame, dark) => {
			try {
				frame.contentDocument?.documentElement.classList.toggle("dark", dark);
			} catch {
				// The preview route is same-origin. If an embedding changes that contract,
				// the parent theme must still toggle without throwing.
			}
		};
		document.addEventListener("click", (event) => {
			if (!event.target.closest("[data-site-theme-toggle]")) return;
			const dark = document.documentElement.classList.toggle("dark");
			try {
				localStorage.setItem("gsxui-theme", dark ? "dark" : "light");
			} catch {
				// storage unavailable (private mode etc.) — the toggle still works
				// for this page view, it just won't persist.
			}
			document.querySelectorAll("iframe[data-site-isolated-preview]").forEach((frame) => syncIsolatedPreviewTheme(frame, dark));
		});
		// A lazy iframe can finish loading after the parent theme changed. Sync it
		// from the resolved parent class at the document boundary as well.
		addEventListener(
			"load",
			(event) => {
				const frame = event.target;
				if (!(frame instanceof HTMLIFrameElement) || !frame.matches("[data-site-isolated-preview]")) return;
				syncIsolatedPreviewTheme(frame, document.documentElement.classList.contains("dark"));
			},
			true,
		);
	</script>
}

// ThemeToggle is the header button: a gsxui ghost icon Button with the Lucide sun-moon icon, marked
// the way gsxui's site marks its toggle (data-site-theme-toggle).
component ThemeToggle() {
	<ui.Button
		variant="ghost"
		size="icon-sm"
		type="button"
		data-site-theme-toggle
		aria-label={locales.NavToggleTheme}
		title={locales.NavToggleTheme}
	>
		<icon.SunMoon/>
	</ui.Button>
}
