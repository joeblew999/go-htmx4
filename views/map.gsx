package views

import (
	"github.com/joeblew999/go-htmx4/locales"
	"github.com/joeblew999/go-htmx4/ui"
	"github.com/joeblew999/go-htmx4/views/gsxkit"
)

// MapPage is gsxkit's map component (views/gsxkit/map.gsx, from go tool gsxkit add map, #164) showing a place the
// way coldter/tide-info's coast map does: a card around where Cloudflare places this request, its pin and popup, a
// button that adds where the reader's browser says they are, and a legend.
component MapPage(p MapProps) {
	<Layout title={locales.MapTitle} description={locales.MapDescription}>
		<h1 class="text-3xl font-semibold tracking-tight">{ locales.MapHeading }</h1>
		<p class="text-muted-foreground">{ locales.MapIntro }</p>
		<ui.Card>
			<ui.CardHeader>
				<ui.CardTitle>{ locales.MapWhereTitle }</ui.CardTitle>
				<ui.CardDescription>
					{ if p.Found {
						{ locales.MapWhereFound(p.Place) }
					} else {
						{ locales.MapWhereUnknown }
					} }
				</ui.CardDescription>
			</ui.CardHeader>
			<ui.CardContent class="flex flex-col gap-4">
				<gsxkit.Map center={p.Center} zoom={p.Zoom} label={locales.MapLabel} locate={true} class="h-96">
					{ if p.Found {
						<gsxkit.MapMarker at={p.At} title={locales.MapCloudflarePin(p.Place) |> text}>
							<p class="font-medium" dir="auto">{ p.Place }</p>
							{ if p.Country != "" {
								<p>{ p.Country |> intl("DisplayNames", "of", "type", "region") }</p>
							} }
							{ if p.Colo != "" {
								<p class="text-muted-foreground">{ locales.MapDataCenter(p.Colo) }</p>
							} }
						</gsxkit.MapMarker>
					} }
				</gsxkit.Map>
				<ul class="flex flex-wrap gap-x-4 gap-y-2 text-sm text-muted-foreground">
					<li class="flex items-center gap-1.5">
						<gsxkit.MapPin class="size-5"/>
						{ locales.MapLegendCloudflare }
					</li>
					<li class="flex items-center gap-1.5">
						<gsxkit.MapPin tone="you" class="size-5"/>
						{ locales.MapLegendYou }
					</li>
				</ul>
			</ui.CardContent>
		</ui.Card>
	</Layout>
}
