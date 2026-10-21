package views

import (
	"github.com/joeblew999/go-htmx4/intl"
	"github.com/joeblew999/go-htmx4/locales"
	"github.com/joeblew999/go-htmx4/ui"
	"github.com/joeblew999/go-htmx4/ui/icon"
)

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
			aria-label={locales.NavPreferences}
			title={locales.NavPreferences}
			hx-get={"/fragments/preferences" |> localize}
			hx-target="#preferences"
			hx-swap="innerHTML"
		>
			<icon.Clock/>
		</ui.Button>
		<ui.DialogContent hideCloseButton={true}>
			<ui.DialogHeader>
				<ui.DialogTitle>{ locales.PreferencesTitle }</ui.DialogTitle>
				<ui.DialogDescription>{ locales.PreferencesDescription }</ui.DialogDescription>
			</ui.DialogHeader>
			<div id="preferences" class="text-sm text-muted-foreground">{ locales.PreferencesLoading }</div>
		</ui.DialogContent>
	</ui.Dialog>
}

// PreferencesForm answers GET /fragments/preferences. A plain POST form: saving redirects back to the page, which
// re-renders with the new <html data-tz data-hc>. The zone list is the browser's own (Intl.supportedValuesOf,
// filled in by intl-browser.mjs), each named in the page's language; without JavaScript it offers automatic and the
// current choice. The clock options are labelled by an example time in each clock, redone by intl-browser.mjs.
component PreferencesForm(returnTo string, connectionZone string) {
	<form method="post" action={"/preferences" |> localize} class="flex flex-col gap-4">
		<input type="hidden" name="return" value={returnTo}/>
		<ui.FieldGroup>
			<ui.Field>
				<ui.FieldLabel for="pref-tz">{ locales.PreferencesTimeZone }</ui.FieldLabel>
				<ui.NativeSelect id="pref-tz" name="tz" class="w-full" data-intl-zones>
					<ui.NativeSelectOption
						value=""
						selected={intl.ChosenTimeZone(ctx) == ""}
						data-intl="browser-zone"
						data-template={locales.PreferencesTimeZoneAuto(slot)}
					>
						{ locales.PreferencesTimeZoneAuto(connectionZone) }
					</ui.NativeSelectOption>
					{ if intl.ChosenTimeZone(ctx) != "" {
						<ui.NativeSelectOption
							value={intl.TimeZone(ctx)}
							selected={true}
							{ intl.Zone(intl.TimeZone(ctx)).Attrs()... }
						>
							{ intl.TimeZone(ctx) |> zone |> plain }
						</ui.NativeSelectOption>
					} }
				</ui.NativeSelect>
			</ui.Field>
			<ui.Field>
				<ui.FieldLabel for="pref-hc">{ locales.PreferencesHourCycle }</ui.FieldLabel>
				<ui.NativeSelect id="pref-hc" name="hc" class="w-full">
					<ui.NativeSelectOption
						value=""
						selected={intl.HourCycle(ctx) == ""}
						{ clockExample(`,"viewerClock":false`).Attrs()... }
						data-template={locales.PreferencesHourCycleAuto(slot)}
					>
						{ hourCycleAuto(ctx) }
					</ui.NativeSelectOption>
					<ui.NativeSelectOption
						value="h12"
						selected={intl.HourCycle(ctx) == "h12"}
						{ clockExample(`,"hourCycle":"h12"`).Attrs()... }
					>
						{ clockExample(`,"hourCycle":"h12"`) |> plain }
					</ui.NativeSelectOption>
					<ui.NativeSelectOption
						value="h23"
						selected={intl.HourCycle(ctx) == "h23"}
						{ clockExample(`,"hourCycle":"h23"`).Attrs()... }
					>
						{ clockExample(`,"hourCycle":"h23"`) |> plain }
					</ui.NativeSelectOption>
				</ui.NativeSelect>
			</ui.Field>
		</ui.FieldGroup>
		<ui.DialogFooter showCloseButton={false}>
			<ui.Button variant="outline" type="button" data-gsxui-dialog-close data-gsxui-slot-dialog-footer-close>
				{ locales.HomeDialogClose }
			</ui.Button>
			<ui.Button type="submit">{ locales.PreferencesSave }</ui.Button>
		</ui.DialogFooter>
	</form>
}
