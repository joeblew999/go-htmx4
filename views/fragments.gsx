package views

import (
	"strings"

	"github.com/joeblew999/go-htmx4/ui"
)

// Greeting answers POST /greet: the main swap into #greeting plus an out-of-band toast appended to the gsxui
// toaster (gsxui toaster docs: hx-swap-oob="beforeend:#gsxui-toaster", no htmx-specific JS).
component Greeting(name string, flavour string, shout bool) {
	<div class="flex items-center gap-2 text-sm">
		<ui.Badge>{ flavour }</ui.Badge>
		{ if shout {
			<strong>HELLO, { strings.ToUpper(name) }!</strong>
		} else {
			<span>Hello, { name }.</span>
		} }
	</div>
	<div hx-swap-oob="beforeend:#gsxui-toaster">
		<ui.Toast toastType="success" title="Greeting rendered" description={"Server-rendered toast for " + name + "."}/>
	</div>
}

// GreetingCleared answers DELETE /greet (sent via hx-action + hx-method). The main content must not be empty:
// htmx 4 leaves the target untouched when a response has only out-of-band content.
component GreetingCleared() {
	<p class="text-sm text-muted-foreground">Cleared.</p>
	<div hx-swap-oob="beforeend:#gsxui-toaster">
		<ui.Toast toastType="info" title="Cleared" description="Sent with hx-action and hx-method delete."/>
	</div>
}

component GreetingError(message string) {
	<ui.FieldError>{ message }</ui.FieldError>
}

component ServerInfoView(info ServerInfo) {
	<dl class="grid grid-cols-[auto_1fr] gap-x-4 gap-y-1">
		<dt class="font-medium text-foreground">Go</dt>
		<dd>{ info.GoVersion }</dd>
		<dt class="font-medium text-foreground">Uptime</dt>
		<dd>{ info.Uptime }</dd>
		<dt class="font-medium text-foreground">Requests</dt>
		<dd>{ info.Requests }</dd>
		<dt class="font-medium text-foreground">Rendered</dt>
		<dd>{ info.Now }</dd>
		{ if info.Note != "" {
			<dt class="font-medium text-foreground">Note</dt>
			<dd>{ info.Note }</dd>
		} }
	</dl>
}

component Stats(stats map[string]int) {
	<ul class="flex flex-col gap-1">
		{ for _, k := range sortedKeys(stats) {
			<li class="flex justify-between">
				<span>{ k }</span>
				<ui.Badge variant="secondary">{ stats[k] }</ui.Badge>
			</li>
		} }
	</ul>
}
