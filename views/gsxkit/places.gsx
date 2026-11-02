package gsxkit

import (
	"github.com/gsxhq/gsx"
	"github.com/joeblew999/go-htmx4/locales"
	"github.com/joeblew999/go-htmx4/ui"
)

// PlaceResult is one localized Nominatim search result.
type PlaceResult struct {
	Name, Country, Region, City, Street string
	CountryCode                         string
	Latitude, Longitude                 string
	OpenURL                             string
}

// PlaceSearchState is the current step, its search results, and selected parents.
type PlaceSearchState struct {
	Step, BackStep, Term string
	Country, CountryCode string
	Region, City, Street string
	Results              []PlaceResult
	Selected             *PlaceResult
	Status               string
}

// Places progressively narrows a place search one submitted level at a time.
component Places(id string, action string, state PlaceSearchState, attrs gsx.Attrs) {
	<section class="flex flex-col gap-4" data-gsxkit-places data-step={state.Step} { attrs... }>
		<h2 class="text-xl font-semibold">{ locales.GsxkitPlacesTitle }</h2>
		<p class="text-sm text-muted-foreground">{ locales.GsxkitPlacesGuide }</p>
		<ol class="grid gap-2 sm:grid-cols-2 lg:grid-cols-4">
			<li class={ "rounded-md border border-border p-3", "border-primary bg-primary/10": state.Step == "country" }>
				<p class="text-sm font-medium">{ locales.GsxkitPlacesCountry }</p>
				{ if state.Country != "" {
					<p class="mt-1 text-sm" dir="auto">{ state.Country }</p>
				} }
			</li>
			<li class={ "rounded-md border border-border p-3", "border-primary bg-primary/10": state.Step == "region" }>
				<p class="text-sm font-medium">{ locales.GsxkitPlacesRegion }</p>
				{ if state.Region != "" {
					<p class="mt-1 text-sm" dir="auto">{ state.Region }</p>
				} }
			</li>
			<li class={ "rounded-md border border-border p-3", "border-primary bg-primary/10": state.Step == "city" }>
				<p class="text-sm font-medium">{ locales.GsxkitPlacesCity }</p>
				{ if state.City != "" {
					<p class="mt-1 text-sm" dir="auto">{ state.City }</p>
				} }
			</li>
			<li class={ "rounded-md border border-border p-3", "border-primary bg-primary/10": state.Step == "street" }>
				<p class="text-sm font-medium">{ locales.GsxkitPlacesStreet }</p>
				{ if state.Street != "" {
					<p class="mt-1 text-sm" dir="auto">{ state.Street }</p>
				} }
			</li>
		</ol>
		{ if state.Step != "done" {
			<form
				class="flex flex-col gap-4"
				hx-get={action}
				hx-target="closest [data-gsxkit-places]"
				hx-swap="outerHTML"
			>
				<input type="hidden" name="step" value={state.Step}/>
				<input type="hidden" name="search" value="true"/>
				<input type="hidden" name="country" value={state.Country}/>
				<input type="hidden" name="country-code" value={state.CountryCode}/>
				<input type="hidden" name="region" value={state.Region}/>
				<input type="hidden" name="city" value={state.City}/>
				<input type="hidden" name="street" value={state.Street}/>
				<ui.FieldGroup>
					<ui.Field>
						<ui.FieldLabel for={f`@{id}-@{state.Step}-term`}>
							{ if state.Step == "country" {
								{ locales.GsxkitPlacesCountry }
							} else if state.Step == "region" {
								{ locales.GsxkitPlacesRegion }
							} else if state.Step == "city" {
								{ locales.GsxkitPlacesCity }
							} else {
								{ locales.GsxkitPlacesStreet }
							} }
						</ui.FieldLabel>
						<ui.Input id={f`@{id}-@{state.Step}-term`} name="term" value={state.Term} required={true}/>
					</ui.Field>
				</ui.FieldGroup>
				<div class="flex flex-wrap items-center gap-3">
					<ui.Button type="submit">{ locales.GsxkitPlacesSearch }</ui.Button>
					<p class="text-sm text-muted-foreground">{ locales.GsxkitPlacesPrivacy }</p>
				</div>
			</form>
		} }
		{ if state.BackStep != "" {
			<form hx-get={action} hx-target="closest [data-gsxkit-places]" hx-swap="outerHTML">
				<input type="hidden" name="step" value={state.BackStep}/>
				<input type="hidden" name="back" value="true"/>
				<input type="hidden" name="country" value={state.Country}/>
				<input type="hidden" name="country-code" value={state.CountryCode}/>
				<input type="hidden" name="region" value={state.Region}/>
				<input type="hidden" name="city" value={state.City}/>
				<input type="hidden" name="street" value={state.Street}/>
				<ui.Button variant="outline" type="submit">{ locales.GsxkitPlacesBack }</ui.Button>
			</form>
		} }
		{ if state.Status == "empty" {
			<ui.Alert>
				<ui.AlertDescription>{ locales.GsxkitPlacesEmpty }</ui.AlertDescription>
			</ui.Alert>
		} else if state.Status == "unavailable" {
			<ui.Alert variant="destructive">
				<ui.AlertDescription>{ locales.GsxkitPlacesUnavailable }</ui.AlertDescription>
			</ui.Alert>
		} else if state.Status == "invalid" {
			<ui.Alert variant="destructive">
				<ui.AlertDescription>{ locales.GsxkitPlacesInvalid }</ui.AlertDescription>
			</ui.Alert>
		} else if state.Status == "limited" {
			<ui.Alert>
				<ui.AlertDescription>{ locales.GsxkitPlacesLimited }</ui.AlertDescription>
			</ui.Alert>
		} else if state.Status == "none" {
			<ui.Alert>
				<ui.AlertDescription>{ locales.GsxkitPlacesNoResults }</ui.AlertDescription>
			</ui.Alert>
		} }
		{ if state.Selected != nil {
			<ui.Alert>
				<ui.AlertTitle>{ locales.GsxkitPlacesSelected }</ui.AlertTitle>
				<ui.AlertDescription dir="auto">
					{ state.Selected.Name }{ " " }{ state.Selected.Street }{ " " }{ state.Selected.City }{ " " }
					{ state.Selected.Region }{ " " }{ state.Selected.Country }
				</ui.AlertDescription>
			</ui.Alert>
		} }
		{ if len(state.Results) > 0 {
			<ul class="grid gap-3">
				{ for _, place := range state.Results {
					<li>
						<ui.Card>
							<ui.CardContent class="flex flex-col gap-2 p-4">
								<p class="font-medium" dir="auto">{ place.Name }</p>
								<p class="text-sm text-muted-foreground" dir="auto">
									{ place.Street }{ " " }{ place.City }{ " " }{ place.Region }{ " " }{ place.Country }
								</p>
								<div class="flex flex-wrap gap-2">
									<form hx-get={action} hx-target="closest [data-gsxkit-places]" hx-swap="outerHTML">
										<input type="hidden" name="step" value={state.Step}/>
										<input type="hidden" name="choose" value="true"/>
										<input type="hidden" name="country" value={state.Country}/>
										<input type="hidden" name="country-code" value={state.CountryCode}/>
										<input type="hidden" name="region" value={state.Region}/>
										<input type="hidden" name="city" value={state.City}/>
										<input type="hidden" name="street" value={state.Street}/>
										<input type="hidden" name="place-name" value={place.Name}/>
										<input type="hidden" name="place-country" value={place.Country}/>
										<input type="hidden" name="place-country-code" value={place.CountryCode}/>
										<input type="hidden" name="place-region" value={place.Region}/>
										<input type="hidden" name="place-city" value={place.City}/>
										<input type="hidden" name="place-street" value={place.Street}/>
										<input type="hidden" name="latitude" value={place.Latitude}/>
										<input type="hidden" name="longitude" value={place.Longitude}/>
										<input type="hidden" name="open-url" value={place.OpenURL}/>
										<ui.Button type="submit" size="sm">
											{ if state.Step == "country" {
												{ locales.GsxkitPlacesChooseCountry }
											} else if state.Step == "region" {
												{ locales.GsxkitPlacesChooseRegion }
											} else if state.Step == "city" {
												{ locales.GsxkitPlacesChooseCity }
											} else {
												{ locales.GsxkitPlacesChooseStreet }
											} }
										</ui.Button>
									</form>
									<ui.Button
										href={place.OpenURL}
										variant="outline"
										size="sm"
										target="_blank"
										rel="noopener noreferrer"
									>
										{ locales.GsxkitPlacesOpenMap }
									</ui.Button>
								</div>
							</ui.CardContent>
						</ui.Card>
					</li>
				} }
			</ul>
		} }
		<p class="text-xs text-muted-foreground">
			<a href="https://www.openstreetmap.org/copyright">{ locales.GsxkitPlacesAttribution }</a>
		</p>
	</section>
}
