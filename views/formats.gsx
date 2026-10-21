package views

import "github.com/joeblew999/go-htmx4/ui"

// Formats is the Intl showcase: every Intl API with examples in the request's language. Each example shows its input
// and options, then two results from the same code (worker/intl.mjs): the Worker's, formatted on the server with
// Cloudflare's V8 Intl, and the reader's browser's, re-formatted live (static/intl.js). They match, except where the
// browser knows more (the reader's time zone) or words things its own way (Firefox, Safari). The Try it card
// round-trips values typed the reader's way through the server.
component Formats() {
	<Layout title={M(ctx).FormatsTitle()} description={M(ctx).FormatsDescription()} path="/formats">
		<div class="flex flex-col gap-2">
			<h1 class="text-3xl font-semibold tracking-tight">{ M(ctx).FormatsTitle() }</h1>
			<p class="text-muted-foreground">{ M(ctx).FormatsIntro() }</p>
		</div>
		<HowItWorks/>
		<LocaleCard facts={Facts(ctx)}/>
		<IntlNav apis={IntlAPIs(ctx)}/>
		<TryIt/>
		{ for _, api := range IntlAPIs(ctx) {
			<IntlAPICard api={api}/>
		} }
	</Layout>
}

// HowItWorks is the page's three-step explanation: formatted on the Worker, refreshed in the browser, sent back plain.
component HowItWorks() {
	<div class="grid gap-4 sm:grid-cols-3">
		<ui.Card data-size="sm">
			<ui.CardHeader>
				<ui.CardTitle>{ M(ctx).FormatsHowServerTitle() }</ui.CardTitle>
				<ui.CardDescription>{ M(ctx).FormatsHowServer() }</ui.CardDescription>
			</ui.CardHeader>
		</ui.Card>
		<ui.Card data-size="sm">
			<ui.CardHeader>
				<ui.CardTitle>{ M(ctx).FormatsHowBrowserTitle() }</ui.CardTitle>
				<ui.CardDescription>{ M(ctx).FormatsHowBrowser() }</ui.CardDescription>
			</ui.CardHeader>
		</ui.Card>
		<ui.Card data-size="sm">
			<ui.CardHeader>
				<ui.CardTitle>{ M(ctx).FormatsHowRoundtripTitle() }</ui.CardTitle>
				<ui.CardDescription>{ M(ctx).FormatsHowRoundtrip() }</ui.CardDescription>
			</ui.CardHeader>
		</ui.Card>
	</div>
}

// IntlNav links every section of the page; it wraps on narrow screens.
component IntlNav(apis []IntlAPI) {
	<nav aria-label={M(ctx).FormatsOnThisPage()} class="flex flex-col gap-2">
		<h2 class="text-sm font-medium text-muted-foreground">{ M(ctx).FormatsOnThisPage() }</h2>
		<ul class="flex flex-wrap gap-2">
			<li>
				<a href="#try" class="inline-flex rounded-md border px-2 py-1 text-sm hover:bg-muted">{ M(ctx).FormatsTryTitle() }</a>
			</li>
			{ for _, api := range apis {
				<li>
					<a
						href={"#" + api.ID}
						class="inline-flex rounded-md border px-2 py-1 font-mono text-xs hover:bg-muted"
						translate="no"
						dir="ltr"
					>
						{ api.Name }
					</a>
				</li>
			} }
		</ul>
	</nav>
}

// IntlAPICard is one Intl API: its name, then each group of examples with a heading and description.
component IntlAPICard(api IntlAPI) {
	<section id={api.ID} class="scroll-mt-4">
		<ui.Card>
			<ui.CardHeader>
				<ui.CardTitle>
					<h2 class="font-mono text-lg" translate="no" dir="ltr">{ api.Name }</h2>
				</ui.CardTitle>
			</ui.CardHeader>
			<ui.CardContent class="flex flex-col gap-6">
				{ for _, g := range api.Groups {
					<div class="flex flex-col gap-2">
						<div class="flex flex-col gap-1">
							<h3 class="font-medium">{ g.Title }</h3>
							<p class="text-sm text-muted-foreground">{ g.Description }</p>
						</div>
						<ul class="flex flex-col divide-y rounded-md border">
							{ for _, r := range g.Rows {
								<IntlExample r={r}/>
							} }
						</ul>
					</div>
				} }
			</ui.CardContent>
		</ui.Card>
	</section>
}

// IntlExample is one example: input and options, then the Worker's result and the browser's.
component IntlExample(r FormatRow) {
	<li class="flex flex-col gap-1.5 px-3 py-2.5">
		<div class="flex flex-wrap items-baseline justify-between gap-x-4 gap-y-1">
			<code dir="ltr" translate="no" class="text-sm font-medium break-all">{ r.Label }</code>
			<code dir="ltr" translate="no" class="text-xs text-muted-foreground break-all">{ r.Options }</code>
		</div>
		<dl class="grid grid-cols-[auto_1fr] items-baseline gap-x-3 gap-y-0.5 text-sm">
			<dt class="text-xs text-muted-foreground">{ M(ctx).FormatsServer() }</dt>
			<dd>
				<bdi data-i18n="intl">{ r.Text(ctx) }</bdi>
			</dd>
			{ if r.Kind != "" {
				<dt class="text-xs text-muted-foreground">{ M(ctx).FormatsBrowser() }</dt>
				<dd>
					<bdi
						data-intl={r.Kind}
						data-value={r.Value}
						{ if r.Opts != "" {
							data-opts={r.Opts}
						} }
						{ if r.End != "" {
							data-end={r.End}
						} }
						{ if r.Unit != "" {
							data-unit={r.Unit}
						} }
					>
						{ r.Text(ctx) }
					</bdi>
				</dd>
			} }
		</dl>
	</li>
}

// TryIt round-trips values typed the reader's way through the Worker's Intl (POST /formats/try).
component TryIt() {
	<section id="try" class="scroll-mt-4">
		<ui.Card>
			<ui.CardHeader>
				<ui.CardTitle>
					<h2 class="text-lg">{ M(ctx).FormatsTryTitle() }</h2>
				</ui.CardTitle>
				<ui.CardDescription>{ M(ctx).FormatsTryDescription() }</ui.CardDescription>
			</ui.CardHeader>
			<ui.CardContent class="flex flex-col gap-6">
				<form hx-post={URL(ctx, "/formats/try")} hx-target="#try-result" class="flex flex-col gap-4">
					<ui.FieldGroup>
						<ui.Field>
							<ui.FieldLabel for="try-number">{ M(ctx).FormatsTryNumber() }</ui.FieldLabel>
							<ui.Input id="try-number" name="number" inputmode="decimal" autocomplete="off" value={TrySample(ctx)}/>
						</ui.Field>
						<ui.Field>
							<ui.FieldLabel for="try-text">{ M(ctx).FormatsTryText() }</ui.FieldLabel>
							<ui.Input id="try-text" name="text" autocomplete="off" value={M(ctx).HomeHeading()}/>
						</ui.Field>
					</ui.FieldGroup>
					<ui.Button type="submit" class="self-start">{ M(ctx).FormatsTrySubmit() }</ui.Button>
				</form>
				<div id="try-result" aria-live="polite">
					<TryResultView res={RoundTrip(ctx, TrySample(ctx), M(ctx).HomeHeading())}/>
				</div>
			</ui.CardContent>
		</ui.Card>
	</section>
}

// TryResultView answers POST /formats/try (and renders the card's first result).
component TryResultView(res TryResult) {
	{ if res.NoRuntime {
		<p class="text-sm text-muted-foreground">{ M(ctx).FormatsTryNoRuntime() }</p>
	} else {
		<div class="grid gap-6 md:grid-cols-2">
			{ if res.Number != "" {
				<div class="flex flex-col gap-2">
					{ if res.NumberOK {
						<p class="text-sm">
							{ M(ctx).FormatsTryParsed() }
							<code dir="ltr" translate="no" class="ms-1 rounded bg-muted px-1.5 py-0.5 font-medium">{ res.Parsed }</code>
						</p>
						<ul class="flex flex-col divide-y rounded-md border">
							{ for _, row := range res.NumberRows {
								<TryRowView row={row} code={true}/>
							} }
						</ul>
					} else {
						<ui.FieldError>{ M(ctx).FormatsTryNotNumber() }</ui.FieldError>
					} }
				</div>
			} }
			{ if len(res.TextRows) > 0 {
				<ul class="flex flex-col divide-y self-start rounded-md border">
					{ for _, row := range res.TextRows {
						<TryRowView row={row} code={false}/>
					} }
				</ul>
			} }
		</div>
	} }
}

component TryRowView(row TryRow, code bool) {
	<li class="flex flex-wrap items-baseline justify-between gap-x-4 gap-y-1 px-3 py-2 text-sm">
		{ if code {
			<code dir="ltr" translate="no" class="text-xs text-muted-foreground">{ row.Label }</code>
		} else {
			<span class="text-muted-foreground">{ row.Label }</span>
		} }
		<bdi data-i18n="intl" class="font-medium">{ row.Text }</bdi>
	</li>
}

component LocaleCard(facts LocaleFacts) {
	<ui.Card>
		<ui.CardHeader>
			<ui.CardTitle translate="no">{ facts.Name }</ui.CardTitle>
		</ui.CardHeader>
		<ui.CardContent class="flex flex-wrap gap-2">
			<ui.Badge variant="secondary" translate="no">{ facts.ID }</ui.Badge>
			<ui.Badge variant="outline" translate="no">{ facts.Maximal }</ui.Badge>
			<ui.Badge variant="outline">{ M(ctx).FormatsDirection(facts.Dir) }</ui.Badge>
			<ui.Badge variant="outline">{ M(ctx).FormatsNumbering(facts.NumberingSystem) }</ui.Badge>
			{ if facts.NativeSystem != "" {
				<ui.Badge variant="outline">{ M(ctx).FormatsNative(facts.NativeSystem) }</ui.Badge>
			} }
			<ui.Badge variant="outline">{ M(ctx).FormatsTimeZone(TimeZone(ctx)) }</ui.Badge>
		</ui.CardContent>
	</ui.Card>
}
