package views

import (
	"time"

	"github.com/joeblew999/go-htmx4/kit/i18n"

	"github.com/joeblew999/go-htmx4/ui"
	"github.com/joeblew999/go-htmx4/ui/icon"
)

// LocalTime is an instant for the viewer, Intl.DateTimeFormat with opts (a JSON object of its options) in the page's
// language: the server's text uses the viewer's chosen zone or the connection's, the browser's its own.
component LocalTime(t time.Time, opts string) {
	<time datetime={isoInstant(t)} data-intl="datetime" data-opts={opts}>
		{ IntlText(ctx, i18n.Call{Kind: "datetime", Value: isoInstant(t), Opts: opts}, utcText(t)) }
	</time>
}

// RelativeTime is an instant relative to now ("3 minutes ago", Intl.RelativeTimeFormat, numeric "auto"), kept
// fresh by intl.js; its tooltip is the date in the viewer's zone.
component RelativeTime(iso string, fallback string) {
	<time datetime={iso} title={fallback} data-intl="relative">
		{ IntlText(ctx, i18n.Call{Kind: "relative", Value: iso}, fallback) }
	</time>
}

// Elapsed is a duration (Intl.DurationFormat, short style), e.g. "1 day, 2 hr, 5 min, 30 sec".
component Elapsed(d time.Duration) {
	<data value={durationJSON(d)} data-intl="duration">
		{ IntlText(ctx, i18n.Call{Kind: "duration", Value: durationJSON(d)}, durationText(d)) }
	</data>
}

// PreferencesDialog is the header's time zone and clock dialog. Its form loads when it opens, and the trigger swaps
// it in with innerHTML: the header nav's inherited outerMorph would replace the target itself.
component PreferencesDialog() {
	<ui.Dialog>
		<ui.Button
			variant="ghost"
			size="icon-sm"
			data-gsxui-slot-dialog-trigger
			aria-haspopup="dialog"
			aria-expanded="false"
			aria-label={M(ctx).NavPreferences()}
			title={M(ctx).NavPreferences()}
			hx-get={URL(ctx, "/fragments/preferences")}
			hx-target="#preferences"
			hx-swap="innerHTML"
		>
			<icon.Clock/>
		</ui.Button>
		<ui.DialogContent hideCloseButton={true}>
			<ui.DialogHeader>
				<ui.DialogTitle>{ M(ctx).PreferencesTitle() }</ui.DialogTitle>
				<ui.DialogDescription>{ M(ctx).PreferencesDescription() }</ui.DialogDescription>
			</ui.DialogHeader>
			<div id="preferences" class="text-sm text-muted-foreground">{ M(ctx).PreferencesLoading() }</div>
		</ui.DialogContent>
	</ui.Dialog>
}

// PreferencesForm answers GET /fragments/preferences. A plain POST form: saving redirects back to the page, which
// re-renders with the new <html data-tz data-hc>. The zone list is the browser's own (Intl.supportedValuesOf,
// filled in by intl.js), each named in the page's language; without JavaScript it offers automatic and the
// current choice. The clock options are labelled by an example time in each clock, also formatted by intl.js.
component PreferencesForm(returnTo string, connectionZone string) {
	<form method="post" action={URL(ctx, "/preferences")} class="flex flex-col gap-4">
		<input type="hidden" name="return" value={returnTo}/>
		<ui.FieldGroup>
			<ui.Field>
				<ui.FieldLabel for="pref-tz">{ M(ctx).PreferencesTimeZone() }</ui.FieldLabel>
				<ui.NativeSelect id="pref-tz" name="tz" class="w-full" data-intl-zones>
					<ui.NativeSelectOption
						value=""
						selected={!TimeZoneChosen(ctx)}
						data-intl="browser-zone"
						data-template={M(ctx).PreferencesTimeZoneAuto(slot)}
					>
						{ M(ctx).PreferencesTimeZoneAuto(connectionZone) }
					</ui.NativeSelectOption>
					{ if TimeZoneChosen(ctx) {
						<ui.NativeSelectOption value={TimeZone(ctx)} selected={true} data-intl="zone" data-value={TimeZone(ctx)}>
							{ IntlText(ctx, i18n.Call{Kind: "zone", Value: TimeZone(ctx)}, TimeZone(ctx)) }
						</ui.NativeSelectOption>
					} }
				</ui.NativeSelect>
			</ui.Field>
			<ui.Field>
				<ui.FieldLabel for="pref-hc">{ M(ctx).PreferencesHourCycle() }</ui.FieldLabel>
				<ui.NativeSelect id="pref-hc" name="hc" class="w-full">
					<ui.NativeSelectOption
						value=""
						selected={HourCycle(ctx) == ""}
						data-intl="datetime"
						data-value="2026-01-01T13:05:00Z"
						data-opts={`{"hour":"numeric","minute":"2-digit","timeZone":"UTC","viewerClock":false}`}
						data-template={M(ctx).PreferencesHourCycleAuto(slot)}
					>
						{ M(ctx).PreferencesHourCycleAuto(IntlText(ctx, i18n.Call{Kind: "datetime", Value: "2026-01-01T13:05:00Z", Opts: `{"hour":"numeric","minute":"2-digit","timeZone":"UTC","viewerClock":false}`}, "13:05")) }
					</ui.NativeSelectOption>
					<ui.NativeSelectOption
						value="h12"
						selected={HourCycle(ctx) == "h12"}
						data-intl="datetime"
						data-value="2026-01-01T13:05:00Z"
						data-opts={`{"hour":"numeric","minute":"2-digit","hourCycle":"h12","timeZone":"UTC"}`}
					>
						{ IntlText(ctx, i18n.Call{Kind: "datetime", Value: "2026-01-01T13:05:00Z", Opts: `{"hour":"numeric","minute":"2-digit","hourCycle":"h12","timeZone":"UTC"}`}, "1:05 PM") }
					</ui.NativeSelectOption>
					<ui.NativeSelectOption
						value="h23"
						selected={HourCycle(ctx) == "h23"}
						data-intl="datetime"
						data-value="2026-01-01T13:05:00Z"
						data-opts={`{"hour":"numeric","minute":"2-digit","hourCycle":"h23","timeZone":"UTC"}`}
					>
						{ IntlText(ctx, i18n.Call{Kind: "datetime", Value: "2026-01-01T13:05:00Z", Opts: `{"hour":"numeric","minute":"2-digit","hourCycle":"h23","timeZone":"UTC"}`}, "13:05") }
					</ui.NativeSelectOption>
				</ui.NativeSelect>
			</ui.Field>
		</ui.FieldGroup>
		<ui.DialogFooter showCloseButton={false}>
			<ui.Button variant="outline" type="button" data-gsxui-dialog-close data-gsxui-slot-dialog-footer-close>
				{ M(ctx).HomeDialogClose() }
			</ui.Button>
			<ui.Button type="submit">{ M(ctx).PreferencesSave() }</ui.Button>
		</ui.DialogFooter>
	</form>
}
