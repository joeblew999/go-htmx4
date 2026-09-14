package views

import (
	"time"

	"github.com/joeblew999/go-htmx4/kit/i18n"
	"github.com/joeblew999/go-htmx4/ui"
	"github.com/joeblew999/go-htmx4/ui/icon"
)

// LocalTime renders an instant for the viewer: text formatted on the server in the request's locale and the
// viewer's time zone and hour cycle, inside <time datetime> (complete for crawlers). Unless the viewer chose
// the zone, it also carries the Intl options and the zone it used, so static/relative-time.js can re-format
// it in the browser's own zone when that differs (plan decision 6).
component LocalTime(t time.Time, opts i18n.DateTimeOptions) {
	<time
		datetime={isoInstant(t)}
		data-i18n="cldr"
		{ if !TimeZoneChosen(ctx) {
			data-local-time={localTimeJSON(ctx, opts)}
			data-time-zone={TimeZone(ctx)}
		} }
	>
		{ FormatDateTime(ctx, t, opts) }
	</time>
}

// PreferencesDialog is the header's time zone and clock dialog. Its form loads when it opens (the zone list is
// ~420 localized names), and the trigger swaps it in with innerHTML: the header nav's inherited outerMorph
// would replace the target itself.
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

// PreferencesForm answers GET /fragments/preferences. A plain POST form: saving redirects back to the page, so
// every date on it re-renders with the new settings (boosted from the header, a full load without htmx).
component PreferencesForm(returnTo string, connectionZone string) {
	<form method="post" action={URL(ctx, "/preferences")} class="flex flex-col gap-4">
		<input type="hidden" name="return" value={returnTo}/>
		<ui.FieldGroup>
			<ui.Field>
				<ui.FieldLabel for="pref-tz">{ M(ctx).PreferencesTimeZone() }</ui.FieldLabel>
				<ui.NativeSelect id="pref-tz" name="tz" class="w-full">
					<ui.NativeSelectOption value="" selected={!TimeZoneChosen(ctx)}>
						{ M(ctx).PreferencesTimeZoneAuto(connectionZone) }
					</ui.NativeSelectOption>
					{ for _, z := range ZoneOptions(ctx, time.Now()) {
						<ui.NativeSelectOption
							value={z.ID}
							selected={TimeZoneChosen(ctx) && z.ID == TimeZone(ctx)}
							data-i18n="cldr"
						>
							{ z.Label }
						</ui.NativeSelectOption>
					} }
				</ui.NativeSelect>
			</ui.Field>
			<ui.Field>
				<ui.FieldLabel for="pref-hc">{ M(ctx).PreferencesHourCycle() }</ui.FieldLabel>
				<ui.NativeSelect id="pref-hc" name="hc" class="w-full">
					{ for _, h := range HourCycleOptions(ctx) {
						<ui.NativeSelectOption value={h.Value} selected={h.Value == HourCycle(ctx).String()} data-i18n="cldr">
							{ h.Label }
						</ui.NativeSelectOption>
					} }
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
