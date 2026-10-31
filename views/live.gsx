package views

import (
	"github.com/joeblew999/go-htmx4/locales"
	"github.com/joeblew999/go-htmx4/ui"
	"github.com/joeblew999/go-htmx4/ui/icon"
	"github.com/joeblew999/go-htmx4/views/gsxkit"
)

// LivePage is live video over MoQ (#156): the reader's own camera, broadcast on a stream of this page view's own
// with a link to send; or, opened from such a link, the stream it names. The video cards are gsxkit's moq component
// (views/gsxkit/moq.gsx, from go tool gsxkit add moq, #164); the page around them is ours.
component LivePage(p LiveProps) {
	<Layout title={locales.LiveTitle} description={locales.LiveDescription}>
		<h1 class="text-3xl font-semibold tracking-tight">{ locales.LiveHeading }</h1>
		<p class="text-muted-foreground">{ locales.LiveIntro }</p>
		// A link someone sent: the broadcast it names, and a way to start one's own (a full load, so the broadcast card
		// starts fresh). Without one: the reader's own camera. The page view's own stream: an idle publisher holds a
		// name no link watches, so it shadows nothing.
		{ if p.Watch != "" {
			<gsxkit.MoqWatch url={p.WatchURL} name={p.WatchName}/>
			<div>
				<ui.Button variant="outline" href={Live{} |> url} hx-boost:inherited="false" data-testid="moq-broadcast-own">
					<icon.Video/>
					{ locales.LiveBroadcastOwn }
				</ui.Button>
			</div>
		} else {
			<gsxkit.MoqBroadcast url={p.PublishURL} name={p.OwnName} link=f`@{p.Origin}@{Live{Stream: p.Own} |> url}`/>
			<ui.Empty class="border">
				<ui.EmptyHeader>
					<ui.EmptyTitle>{ locales.LiveNoStreamTitle }</ui.EmptyTitle>
					<ui.EmptyDescription>{ locales.LiveNoStreamDescription }</ui.EmptyDescription>
				</ui.EmptyHeader>
			</ui.Empty>
		} }
	</Layout>
}
