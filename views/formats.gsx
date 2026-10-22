package views

import (
	"strings"

	"github.com/gsxhq/gsx"
	"github.com/joeblew999/go-htmx4/intl"
	"github.com/joeblew999/go-htmx4/locales"
	"github.com/joeblew999/go-htmx4/ui"
)

// Formats is the Intl showcase: every Intl API with examples in the request's language. Each example shows its input
// and options, then two results from the same call (intl/js/intl.mjs): the Worker's, formatted on the server with
// Cloudflare's V8 Intl, and the reader's browser's, redone live (intl-browser.mjs). They match, except where the
// browser knows more (the reader's time zone) or words things its own way (Firefox, Safari). The Try it card
// round-trips values typed the reader's way through the server.
component Formats(conn Connection) {
	<Layout title={locales.FormatsTitle} description={locales.FormatsDescription} path="/formats">
		<div class="flex flex-col gap-2">
			<h1 class="text-3xl font-semibold tracking-tight">{ locales.FormatsTitle }</h1>
			<p class="text-muted-foreground">{ locales.FormatsIntro }</p>
		</div>
		<HowItWorks/>
		<LocaleCard/>
		<RequestCard conn={conn}/>
		<IntlNav/>
		<TryIt/>
		<IntlAPIs/>
	</Layout>
}

// HowItWorks is the page's three-step explanation: formatted on the Worker, refreshed in the browser, sent back plain.
component HowItWorks() {
	<div class="grid gap-4 sm:grid-cols-3">
		<ui.Card data-size="sm">
			<ui.CardHeader>
				<ui.CardTitle>{ locales.FormatsHowServerTitle }</ui.CardTitle>
				<ui.CardDescription>{ locales.FormatsHowServer }</ui.CardDescription>
			</ui.CardHeader>
		</ui.Card>
		<ui.Card data-size="sm">
			<ui.CardHeader>
				<ui.CardTitle>{ locales.FormatsHowBrowserTitle }</ui.CardTitle>
				<ui.CardDescription>{ locales.FormatsHowBrowser }</ui.CardDescription>
			</ui.CardHeader>
		</ui.Card>
		<ui.Card data-size="sm">
			<ui.CardHeader>
				<ui.CardTitle>{ locales.FormatsHowRoundtripTitle }</ui.CardTitle>
				<ui.CardDescription>{ locales.FormatsHowRoundtrip }</ui.CardDescription>
			</ui.CardHeader>
		</ui.Card>
	</div>
}

// IntlNav links every section of the page; it wraps on narrow screens.
component IntlNav() {
	<nav aria-label={locales.FormatsOnThisPage} class="flex flex-col gap-2">
		<h2 class="text-sm font-medium text-muted-foreground">{ locales.FormatsOnThisPage }</h2>
		<ul class="flex flex-wrap gap-2">
			<li>
				<a href="#try" class="inline-flex rounded-md border px-2 py-1 text-sm hover:bg-muted">{ locales.FormatsTryTitle }</a>
			</li>
			{ for _, name := range intlAPINames {
				<li>
					<a
						href=f`#@{apiID(name)}`
						class="inline-flex rounded-md border px-2 py-1 font-mono text-xs hover:bg-muted"
						translate="no"
						dir="ltr"
					>
						{ name }
					</a>
				</li>
			} }
		</ul>
	</nav>
}

// IntlAPIs is every Intl API with its examples, each a call written here with intl's filters on formats.go's data.
component IntlAPIs() {
	<IntlAPI name="Intl.DateTimeFormat">
		<Examples title={locales.FormatsDatesTitle} description={locales.FormatsDatesDescription}>
			{ for _, o := range dateOptions {
				<Example in={intl.ISO(sampleInstant)} v={sampleInstant |> date(o...)}/>
			} }
		</Examples>
		<Examples title={locales.FormatsZonesTitle} description={locales.FormatsZonesDescription}>
			{ for _, z := range zoneSamples {
				<Example
					in={z.In}
					v={
						sampleInstant |> date(append(pairs("hour", "numeric", "minute", "2-digit", "timeZone", z.In), z.Opts...)...)
					}
				/>
			} }
		</Examples>
		<Examples title={locales.FormatsRangesTitle} description={locales.FormatsRangesDescription}>
			{ for _, r := range rangeEnds {
				<Example
					in={intl.ISO(sampleInstant) + " – " + intl.ISO(r.End)}
					v={sampleInstant |> dateRange(r.End, r.Opts...)}
				/>
			} }
		</Examples>
	</IntlAPI>
	<IntlAPI name="Intl.RelativeTimeFormat">
		<Examples title={locales.FormatsRelativeTitle} description={locales.FormatsRelativeDescription}>
			{ for _, r := range relativeSamples {
				<Example in={r.N + " " + r.Unit} v={intl.Args{r.N, r.Unit} |> intl("RelativeTimeFormat", "", r.Opts...)}/>
			} }
		</Examples>
	</IntlAPI>
	<IntlAPI name="Intl.NumberFormat">
		<Examples title={locales.FormatsNumbersTitle} description={locales.FormatsNumbersDescription}>
			{ for _, n := range numberSamples {
				<Example in={n.In} v={n.In |> number(n.Opts...)}/>
			} }
			{ for _, r := range numberRanges {
				<Example in={r.In()} v={intl.Args{r.From, r.To} |> intl("NumberFormat", "formatRange", r.Opts...)}/>
			} }
		</Examples>
		<Examples title={locales.FormatsCurrencyTitle} description={locales.FormatsCurrencyDescription(showcaseCurrency)}>
			{ for _, c := range currencySamples {
				<Example in={c.In} v={c.In |> number(c.Opts...)}/>
			} }
		</Examples>
		<Examples title={locales.FormatsUnitsTitle} description={locales.FormatsUnitsDescription}>
			{ for _, u := range unitSamples {
				<Example in={u.In} v={u.In |> number(u.Opts...)}/>
			} }
		</Examples>
	</IntlAPI>
	<IntlAPI name="Intl.PluralRules">
		<Examples title={locales.FormatsPluralsTitle} description={locales.FormatsPluralsDescription}>
			<Example in="pluralCategories" v={intl.Args{} |> intl("PluralRules", "resolvedOptions", "type", "cardinal")}/>
			<Example in="pluralCategories" v={intl.Args{} |> intl("PluralRules", "resolvedOptions", "type", "ordinal")}/>
			{ for _, n := range pluralSamples {
				<Example in={n.In} v={n.In |> intl("PluralRules", "select", n.Opts...)}/>
			} }
			<Example in="1 – 5" v={intl.Args{1, 5} |> intl("PluralRules", "selectRange")}/>
		</Examples>
	</IntlAPI>
	<IntlAPI name="Intl.ListFormat">
		<Examples title={locales.FormatsListsTitle} description={locales.FormatsListsDescription}>
			{ for _, l := range listSamples {
				<Example in={strings.Join(listWords(ctx)[:l.N], " · ")} v={listWords(ctx)[:l.N] |> list(l.Opts...)}/>
			} }
		</Examples>
	</IntlAPI>
	<IntlAPI name="Intl.DurationFormat">
		<Examples title={locales.FormatsDurationsTitle} description={locales.FormatsDurationsDescription}>
			{ for _, o := range durationOptions {
				<Example in="1d 2h 5m 30s" v={sampleDuration |> duration(o...)}/>
			} }
		</Examples>
	</IntlAPI>
	<IntlAPI name="Intl.DisplayNames">
		<Examples title={locales.FormatsDisplayNamesTitle} description={locales.FormatsDisplayNamesDescription}>
			{ for _, n := range displayNameSamples {
				<Example in={n.In} v={n.In |> intl("DisplayNames", "of", n.Opts...)}/>
			} }
		</Examples>
	</IntlAPI>
	<IntlAPI name="Intl.Collator">
		<Examples title={locales.FormatsCollationTitle} description={locales.FormatsCollationDescription}>
			{ for _, c := range comparisons {
				<Example in={c.From + " · " + c.To} v={intl.Args{c.From, c.To} |> intl("Collator", "compare", c.Opts...)}/>
			} }
		</Examples>
	</IntlAPI>
	<IntlAPI name="Intl.Segmenter">
		<Examples title={locales.FormatsSegmentsTitle} description={locales.FormatsSegmentsDescription}>
			{ for _, s := range segmentSamples {
				<Example in={s.In} v={s.In |> intl("Segmenter", "segment", s.Opts...)}/>
			} }
		</Examples>
	</IntlAPI>
	<IntlAPI name="Intl.Locale">
		<Examples title={locales.FormatsLocaleTitle} description={locales.FormatsLocaleDescription}>
			{ for _, m := range localeMethods {
				<Example in={m + "()"} v={intl.Args{} |> intl("Locale", m)}/>
			} }
		</Examples>
	</IntlAPI>
	<IntlAPI name="Intl.supportedValuesOf">
		<Examples title={locales.FormatsSupportedTitle} description={locales.FormatsSupportedDescription}>
			{ for _, key := range supportedKeys {
				<Example in={key} v={key |> intl("Intl", "supportedValuesOf")}/>
			} }
		</Examples>
	</IntlAPI>
}

// IntlAPI is one Intl API's card: its name, then its groups of examples.
component IntlAPI(name string, children gsx.Node) {
	<section id={apiID(name)} class="scroll-mt-4">
		<ui.Card>
			<ui.CardHeader>
				<ui.CardTitle>
					<h2 class="font-mono text-lg" translate="no" dir="ltr">{ name }</h2>
				</ui.CardTitle>
			</ui.CardHeader>
			<ui.CardContent class="flex flex-col gap-6">
				{ children }
			</ui.CardContent>
		</ui.Card>
	</section>
}

// Examples is a group of examples under a heading and a description.
component Examples(title intl.Msg, description intl.Msg, children gsx.Node) {
	<div class="flex flex-col gap-2">
		<div class="flex flex-col gap-1">
			<h3 class="font-medium">{ title }</h3>
			<p class="text-sm text-muted-foreground">{ description }</p>
		</div>
		<ul class="flex flex-col divide-y rounded-md border">
			{ children }
		</ul>
	</div>
}

// Example is one call: its input and options, then the Worker's result and the browser's (the same call, redone in
// the viewer's zone).
component Example(in string, v intl.Value) {
	<li class="flex flex-col gap-1.5 px-3 py-2.5">
		<div class="flex flex-wrap items-baseline justify-between gap-x-4 gap-y-1">
			<code dir="ltr" translate="no" class="text-sm font-medium break-all">{ in }</code>
			<code dir="ltr" translate="no" class="text-xs text-muted-foreground break-all">{ callLabel(v) }</code>
		</div>
		<dl class="grid grid-cols-[auto_1fr] items-baseline gap-x-3 gap-y-0.5 text-sm">
			<dt class="text-xs text-muted-foreground">{ locales.FormatsServer }</dt>
			<dd class="break-all">
				<bdi data-intl-text>{ v |> plain }</bdi>
			</dd>
			<dt class="text-xs text-muted-foreground">{ locales.FormatsBrowser }</dt>
			<dd class="break-all">
				<bdi>{ v }</bdi>
			</dd>
		</dl>
	</li>
}

// TryIt round-trips values typed the reader's way through the Worker's Intl (POST /formats/try).
component TryIt() {
	<section id="try" class="scroll-mt-4">
		<ui.Card>
			<ui.CardHeader>
				<ui.CardTitle>
					<h2 class="text-lg">{ locales.FormatsTryTitle }</h2>
				</ui.CardTitle>
				<ui.CardDescription>{ locales.FormatsTryDescription }</ui.CardDescription>
			</ui.CardHeader>
			<ui.CardContent class="flex flex-col gap-6">
				<form hx-post={"/formats/try" |> localize} hx-target="#try-result" class="flex flex-col gap-4">
					<ui.FieldGroup>
						<ui.Field>
							<ui.FieldLabel for="try-number">{ locales.FormatsTryNumber }</ui.FieldLabel>
							<ui.Input id="try-number" name="number" inputmode="decimal" autocomplete="off" value={TrySample(ctx)}/>
						</ui.Field>
						<ui.Field>
							<ui.FieldLabel for="try-text">{ locales.FormatsTryText }</ui.FieldLabel>
							<ui.Input id="try-text" name="text" autocomplete="off" value={locales.HomeHeading}/>
						</ui.Field>
					</ui.FieldGroup>
					<ui.Button type="submit" class="self-start">{ locales.FormatsTrySubmit }</ui.Button>
				</form>
				<div id="try-result" aria-live="polite">
					<TryResultView res={RoundTrip(ctx, TrySample(ctx), TryText(ctx))}/>
				</div>
			</ui.CardContent>
		</ui.Card>
	</section>
}

// TryResultView answers POST /formats/try (and renders the card's first result).
component TryResultView(res TryResult) {
	<div class="grid gap-6 md:grid-cols-2">
		{ if res.Number != "" {
			<div class="flex flex-col gap-2">
				{ if res.NumberOK {
					<p class="text-sm">
						{ locales.FormatsTryParsed }
						<code dir="ltr" translate="no" class="ms-1 rounded bg-muted px-1.5 py-0.5 font-medium">{ res.Parsed }</code>
					</p>
					<ul class="flex flex-col divide-y rounded-md border">
						{ for _, row := range res.NumberRows {
							<TryRowView row={row} code={true}/>
						} }
					</ul>
				} else {
					<ui.FieldError>{ locales.FormatsTryNotNumber }</ui.FieldError>
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
}

component TryRowView(row TryRow, code bool) {
	<li class="flex flex-wrap items-baseline justify-between gap-x-4 gap-y-1 px-3 py-2 text-sm">
		{ if code {
			<code dir="ltr" translate="no" class="text-xs text-muted-foreground">{ row.Label }</code>
		} else {
			<span class="text-muted-foreground">{ row.Label }</span>
		} }
		<bdi data-intl-text class="font-medium">{ row.Text }</bdi>
	</li>
}

// LocaleCard is what Intl.Locale says of the request's locale, and the zone the page formats in.
component LocaleCard() {
	<ui.Card>
		<ui.CardHeader>
			<ui.CardTitle translate="no">{ intl.Current(ctx).Name() }</ui.CardTitle>
		</ui.CardHeader>
		<ui.CardContent class="flex flex-wrap gap-2">
			<ui.Badge variant="secondary" translate="no">{ intl.Current(ctx).ID() }</ui.Badge>
			<ui.Badge variant="outline" translate="no">{ intl.Current(ctx).Maximal() }</ui.Badge>
			<ui.Badge variant="outline">{ locales.FormatsDirection(intl.Current(ctx).Dir()) }</ui.Badge>
			<ui.Badge variant="outline">{ locales.FormatsTimeZone(intl.TimeZone(ctx)) }</ui.Badge>
		</ui.CardContent>
	</ui.Card>
}

// RequestCard shows how this page chose the reader's language and time zone: everything intl.Site had (the address,
// their browser's languages, their cookies, Cloudflare's view of their connection) and what won (intl.DecisionOf).
component RequestCard(conn Connection) {
	<section id="request" class="scroll-mt-4">
		<ui.Card>
			<ui.CardHeader>
				<ui.CardTitle>
					<h2 class="text-lg">{ locales.FormatsRequestTitle }</h2>
				</ui.CardTitle>
				<ui.CardDescription>{ locales.FormatsRequestDescription }</ui.CardDescription>
			</ui.CardHeader>
			<ui.CardContent>
				<ui.Table>
					<ui.TableBody>
						<FactGroup title={locales.FormatsRequestLanguage}/>
						<Fact label={locales.FormatsRequestPageLanguage}>
							<span translate="no">{ intl.Current(ctx).Name() }</span>{ " " }
							<code translate="no">{ intl.Current(ctx).ID() }</code>
							<div class="text-xs text-muted-foreground">{ languageFrom(intl.DecisionOf(ctx)) }</div>
						</Fact>
						<Fact label={locales.FormatsRequestBrowserAsks}>
							{ if a := intl.DecisionOf(ctx).AcceptLanguage; a != "" {
								<code class="break-all" translate="no" dir="ltr">{ a }</code>
							} else {
								{ locales.FormatsRequestNothing }
							} }
						</Fact>
						<Fact label={locales.FormatsRequestLastViewed}>
							{ if v := intl.DecisionOf(ctx).LastViewed; v != "" {
								<code translate="no">{ v }</code>
							} else {
								{ locales.FormatsRequestFirstVisit }
							} }
						</Fact>
						<FactGroup title={locales.FormatsRequestZone}/>
						<Fact label={locales.FormatsRequestPageZone}>
							{ intl.Zone(intl.TimeZone(ctx)) }
							<div class="text-xs text-muted-foreground">{ zoneFrom(intl.DecisionOf(ctx)) }</div>
						</Fact>
						<Fact label={locales.FormatsRequestChosen}>
							{ if z := intl.DecisionOf(ctx).ChosenZone; z != "" {
								<code translate="no">{ z }</code>
							} else {
								{ locales.FormatsRequestNotChosen }
							} }
						</Fact>
						<Fact label={locales.FormatsRequestBrowserReported}>
							{ if z := intl.DecisionOf(ctx).BrowserZone; z != "" {
								<code translate="no">{ z }</code>
							} else {
								{ locales.FormatsRequestNotReported }
							} }
						</Fact>
						<Fact label={locales.FormatsRequestBrowserNow}>
							<code data-intl="browser-zone">{ locales.FormatsRequestNeedsScript }</code>
						</Fact>
						<Fact label={locales.FormatsRequestConnection}>
							{ if conn.Timezone != "" {
								<code translate="no">{ conn.Timezone }</code>
								{ if conn.Country != "" {
									<div class="text-muted-foreground">
										{ locales.FormatsRequestWhere(conn.City, countryName(conn.Country), conn.Colo) }
									</div>
								} }
							} else {
								{ locales.FormatsRequestNoConnection }
							} }
						</Fact>
						<Fact label={locales.FormatsRequestClock}>{ clockOf(ctx) }</Fact>
					</ui.TableBody>
				</ui.Table>
			</ui.CardContent>
		</ui.Card>
	</section>
}

// FactGroup heads a group of RequestCard's rows.
component FactGroup(title intl.Msg) {
	<ui.TableRow>
		<ui.TableHead colspan="2" class="pt-4 text-foreground">{ title }</ui.TableHead>
	</ui.TableRow>
}

// Fact is one of RequestCard's rows: what it is, and its value (with what decided it, where something did).
component Fact(label intl.Msg, children gsx.Node) {
	<ui.TableRow>
		<ui.TableHead scope="row" class="w-2/5 align-top font-normal whitespace-normal text-muted-foreground">
			{ label }
		</ui.TableHead>
		<ui.TableCell class="align-top whitespace-normal">
			{ children }
		</ui.TableCell>
	</ui.TableRow>
}
