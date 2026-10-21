package views

import (
	"github.com/joeblew999/go-htmx4/locales"

	"github.com/joeblew999/go-htmx4/ui"
)

// Greeting answers POST /greet: the main swap into #greeting plus an out-of-band toast appended to the gsxui
// toaster (gsxui toaster docs: hx-swap-oob="beforeend:#gsxui-toaster", no htmx-specific JS). The name is user
// input: the message renderer isolates its direction inside the sentence, and CSS uppercases it the page's language's
// way (Turkish i → İ).
component Greeting(name string, flavour string, shout bool) {
	<div class="flex items-center gap-2 text-sm">
		<ui.Badge translate="no">{ flavour }</ui.Badge>
		{ if shout {
			<strong class="uppercase">{ locales.GreetShout(name) }</strong>
		} else {
			<span>{ locales.GreetHello(name) }</span>
		} }
	</div>
	<div hx-swap-oob="beforeend:#gsxui-toaster">
		<ui.Toast
			toastType="success"
			title={locales.GreetToastTitle |> text}
			description={locales.GreetToastDescription(name) |> text}
		/>
	</div>
}

// GreetingCleared answers DELETE /greet (sent via hx-action + hx-method). The main content must not be empty:
// htmx 4 leaves the target untouched when a response has only out-of-band content.
component GreetingCleared() {
	<p class="text-sm text-muted-foreground">{ locales.GreetCleared }</p>
	<div hx-swap-oob="beforeend:#gsxui-toaster">
		<ui.Toast
			toastType="info"
			title={locales.GreetClearedToastTitle |> text}
			description={locales.GreetClearedToastDescription |> text}
		/>
	</div>
}

// RateLimited answers a board write over the per-connection limit (429). The posts use hx-swap="none", so
// the out-of-band toast is the whole response.
component RateLimited() {
	<div hx-swap-oob="beforeend:#gsxui-toaster">
		<ui.Toast
			toastType="error"
			title={locales.BoardSlowTitle |> text}
			description={locales.BoardSlowDescription |> text}
		/>
	</div>
}

component GreetingError() {
	<ui.FieldError>{ locales.GreetEmpty }</ui.FieldError>
}

component ServerInfoView(info ServerInfo) {
	<dl class="grid grid-cols-[auto_1fr] gap-x-4 gap-y-1">
		<dt class="font-medium text-foreground">{ locales.ServerInfoGo }</dt>
		<dd translate="no">{ info.GoVersion }</dd>
		<dt class="font-medium text-foreground">{ locales.ServerInfoUptime }</dt>
		<dd>
			{ info.Uptime |> duration }
		</dd>
		<dt class="font-medium text-foreground">{ locales.ServerInfoRequests }</dt>
		<dd>{ info.Requests |> number }</dd>
		<dt class="font-medium text-foreground">{ locales.ServerInfoRendered }</dt>
		<dd>
			{ info.Now |> date(`{"dateStyle":"full","timeStyle":"long"}`) }
		</dd>
		{ if info.Note != "" {
			<dt class="font-medium text-foreground">{ locales.ServerInfoNote }</dt>
			<dd>{ locales.ServerInfoWorkersNote }</dd>
		} }
	</dl>
}

component Stats(stats map[string]int) {
	<ul class="flex flex-col gap-1" translate="no">
		{ for _, k := range sortedKeys(stats) {
			<li class="flex justify-between">
				<span>{ k }</span>
				<ui.Badge variant="secondary">{ stats[k] |> number }</ui.Badge>
			</li>
		} }
	</ul>
}
