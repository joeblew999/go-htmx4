package views

import (
	"strings"

	"github.com/joeblew999/go-htmx4/kit/i18n"
	"github.com/joeblew999/go-htmx4/ui"
)

// Greeting answers POST /greet: the main swap into #greeting plus an out-of-band toast appended to the gsxui
// toaster (gsxui toaster docs: hx-swap-oob="beforeend:#gsxui-toaster", no htmx-specific JS). The name is user
// input: bdi isolates its direction inside the sentence.
component Greeting(name string, flavour string, shout bool) {
	<div class="flex items-center gap-2 text-sm">
		<ui.Badge translate="no">{ flavour }</ui.Badge>
		{ if shout {
			<strong>{ M(ctx).GreetShout(Isolate(Loc(ctx).Dir(), strings.ToUpper(name))) }</strong>
		} else {
			<span>{ M(ctx).GreetHello(Isolate(Loc(ctx).Dir(), name)) }</span>
		} }
	</div>
	<div hx-swap-oob="beforeend:#gsxui-toaster">
		<ui.Toast
			toastType="success"
			title={M(ctx).GreetToastTitle()}
			description={M(ctx).GreetToastDescription(Isolate(Loc(ctx).Dir(), name))}
		/>
	</div>
}

// GreetingCleared answers DELETE /greet (sent via hx-action + hx-method). The main content must not be empty:
// htmx 4 leaves the target untouched when a response has only out-of-band content.
component GreetingCleared() {
	<p class="text-sm text-muted-foreground">{ M(ctx).GreetCleared() }</p>
	<div hx-swap-oob="beforeend:#gsxui-toaster">
		<ui.Toast
			toastType="info"
			title={M(ctx).GreetClearedToastTitle()}
			description={M(ctx).GreetClearedToastDescription()}
		/>
	</div>
}

// RateLimited answers a board write over the per-connection limit (429). The posts use hx-swap="none", so
// the out-of-band toast is the whole response.
component RateLimited() {
	<div hx-swap-oob="beforeend:#gsxui-toaster">
		<ui.Toast toastType="error" title={M(ctx).BoardSlowTitle()} description={M(ctx).BoardSlowDescription()}/>
	</div>
}

component GreetingError() {
	<ui.FieldError>{ M(ctx).GreetEmpty() }</ui.FieldError>
}

component ServerInfoView(info ServerInfo) {
	<dl class="grid grid-cols-[auto_1fr] gap-x-4 gap-y-1">
		<dt class="font-medium text-foreground">{ M(ctx).ServerInfoGo() }</dt>
		<dd translate="no">{ info.GoVersion }</dd>
		<dt class="font-medium text-foreground">{ M(ctx).ServerInfoUptime() }</dt>
		<dd data-i18n="cldr">{ Elapsed(ctx, info.Uptime) }</dd>
		<dt class="font-medium text-foreground">{ M(ctx).ServerInfoRequests() }</dt>
		<dd>{ Num(ctx, info.Requests) }</dd>
		<dt class="font-medium text-foreground">{ M(ctx).ServerInfoRendered() }</dt>
		<dd>
			<LocalTime t={info.Now} opts={i18n.DateTimeOptions{DateStyle: i18n.FullStyle, TimeStyle: i18n.LongStyle}}/>
		</dd>
		{ if info.Note != "" {
			<dt class="font-medium text-foreground">{ M(ctx).ServerInfoNote() }</dt>
			<dd>{ M(ctx).ServerInfoWorkersNote() }</dd>
		} }
	</dl>
}

component Stats(stats map[string]int) {
	<ul class="flex flex-col gap-1" translate="no">
		{ for _, k := range sortedKeys(stats) {
			<li class="flex justify-between">
				<span>{ k }</span>
				<ui.Badge variant="secondary">{ Num(ctx, int64(stats[k])) }</ui.Badge>
			</li>
		} }
	</ul>
}
