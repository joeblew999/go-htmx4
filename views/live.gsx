package views

import (
	"github.com/joeblew999/go-htmx4/locales"
	"github.com/joeblew999/go-htmx4/ui"
)

// LivePage is live video over MoQ (#156): the reader's own camera, broadcast on a stream of this page view's own
// with a link to send, and the stream a link they opened names. The media is @moq's elements (<moq-publish>,
// <moq-watch>, vendored into /static/moq.js); everything around them is gsxui.
//
// Every label for every state is rendered here, translated; /static/moq.js only sets the section's data-state (and
// data-paused, data-muted, data-error), and Tailwind's group-data variants show the one that applies. The script is
// the page's own: the nav reaches this page with a full load, not a boosted swap.
component LivePage(p LiveProps) {
	<Layout title={locales.LiveTitle} description={locales.LiveDescription}>
		<h1 class="text-3xl font-semibold tracking-tight">{ locales.LiveHeading }</h1>
		<p class="text-muted-foreground">{ locales.LiveIntro }</p>
		{ if p.Watch != "" {
			<LiveWatch url={p.WatchURL} name={p.WatchName}/>
		} }
		<LiveBroadcast url={p.PublishURL} name={p.OwnName} share={Live{Stream: p.Own}} origin={p.Origin}/>
		{ if p.Watch == "" {
			<LiveWatch url={p.WatchURL} name={p.WatchName}/>
			<ui.Empty class="border">
				<ui.EmptyHeader>
					<ui.EmptyTitle>{ locales.LiveNoStreamTitle }</ui.EmptyTitle>
					<ui.EmptyDescription>{ locales.LiveNoStreamDescription }</ui.EmptyDescription>
				</ui.EmptyHeader>
			</ui.Empty>
		} }
		<script type="module" src="/static/moq.js"></script>
	</Layout>
}

// LiveWatch plays the broadcast. It starts muted: browsers only play sound after a click.
component LiveWatch(url string, name string) {
	<section
		class="group/moq"
		data-moq="watch"
		data-state="connecting"
		data-paused="false"
		data-muted="true"
	>
		<ui.Card>
			<ui.CardHeader>
				<ui.CardTitle>{ locales.LiveWatchTitle }</ui.CardTitle>
				<ui.CardDescription>{ locales.LiveWatchDescription }</ui.CardDescription>
				<ui.CardAction role="status" aria-live="polite" data-testid="moq-watch-state">
					<ui.Badge variant="secondary" class="hidden group-data-[state=connecting]/moq:inline-flex">
						{ locales.LiveStateConnecting }
					</ui.Badge>
					<ui.Badge variant="secondary" class="hidden group-data-[state=waiting]/moq:inline-flex">
						{ locales.LiveStateWaiting }
					</ui.Badge>
					<ui.Badge class="hidden group-data-[state=playing]/moq:inline-flex">{ locales.LiveStatePlaying }</ui.Badge>
					<ui.Badge variant="secondary" class="hidden group-data-[state=paused]/moq:inline-flex">
						{ locales.LiveStatePaused }
					</ui.Badge>
					<ui.Badge variant="secondary" class="hidden group-data-[state=reconnecting]/moq:inline-flex">
						{ locales.LiveStateReconnecting }
					</ui.Badge>
					<ui.Badge variant="outline" class="hidden group-data-[state=ended]/moq:inline-flex">
						{ locales.LiveStateEnded }
					</ui.Badge>
					<ui.Badge variant="destructive" class="hidden group-data-[state=unsupported]/moq:inline-flex">
						{ locales.LiveStateUnsupported }
					</ui.Badge>
				</ui.CardAction>
			</ui.CardHeader>
			<ui.CardContent>
				<div class="aspect-video overflow-hidden rounded-lg bg-muted" data-moq-frame>
					// announced="false": subscribe by name. Cloudflare's draft-14 relays don't answer namespace subscriptions, so a
					// watcher waiting for the broadcast to be announced would wait forever (#156).
					<moq-watch url={url} name={name} muted announced="false" class="block size-full">
						<canvas class="size-full object-contain" aria-label={locales.LiveWatchTitle |> text}></canvas>
					</moq-watch>
				</div>
				// DIAGNOSTICS on screen (#156): what the watcher's element reports, filled in by /static/moq.js.
				<p class="mt-3 whitespace-pre-wrap break-all font-mono text-xs text-muted-foreground" dir="ltr" translate="no" data-moq-debug></p>
			</ui.CardContent>
			<ui.CardFooter class="flex-wrap gap-2">
				<ui.Button variant="outline" size="sm" data-moq-action="pause" class="group-data-[paused=true]/moq:hidden">
					{ locales.LivePause }
				</ui.Button>
				<ui.Button
					variant="outline"
					size="sm"
					data-moq-action="play"
					class="hidden group-data-[paused=true]/moq:inline-flex"
				>
					{ locales.LivePlay }
				</ui.Button>
				<ui.Button variant="outline" size="sm" data-moq-action="unmute" class="group-data-[muted=false]/moq:hidden">
					{ locales.LiveUnmute }
				</ui.Button>
				<ui.Button
					variant="outline"
					size="sm"
					data-moq-action="mute"
					class="hidden group-data-[muted=false]/moq:inline-flex"
				>
					{ locales.LiveMute }
				</ui.Button>
				<ui.Button variant="outline" size="sm" data-moq-action="fullscreen">{ locales.LiveFullscreen }</ui.Button>
			</ui.CardFooter>
		</ui.Card>
	</section>
}

// LiveBroadcast sends the reader's camera and microphone to the relay. Nothing is captured until they press start.
component LiveBroadcast(url string, name string, share Live, origin string) {
	<section class="group/moq" data-moq="broadcast" data-state="idle" data-muted="false">
		<ui.Card>
			<ui.CardHeader>
				<ui.CardTitle>{ locales.LiveBroadcastTitle }</ui.CardTitle>
				<ui.CardDescription>{ locales.LiveBroadcastDescription }</ui.CardDescription>
				<ui.CardAction role="status" aria-live="polite" data-testid="moq-broadcast-state">
					<ui.Badge variant="outline" class="hidden group-data-[state=idle]/moq:inline-flex">
						{ locales.LiveStateIdle }
					</ui.Badge>
					<ui.Badge variant="secondary" class="hidden group-data-[state=requesting]/moq:inline-flex">
						{ locales.LiveStateRequesting }
					</ui.Badge>
					<ui.Badge variant="secondary" class="hidden group-data-[state=connecting]/moq:inline-flex">
						{ locales.LiveStateConnecting }
					</ui.Badge>
					<ui.Badge variant="destructive" class="hidden group-data-[state=live]/moq:inline-flex">
						{ locales.LiveStateOnAir }
					</ui.Badge>
					<ui.Badge variant="secondary" class="hidden group-data-[state=reconnecting]/moq:inline-flex">
						{ locales.LiveStateReconnecting }
					</ui.Badge>
					<ui.Badge variant="destructive" class="hidden group-data-[state=error]/moq:inline-flex">
						{ locales.LiveStateFailed }
					</ui.Badge>
					<ui.Badge variant="destructive" class="hidden group-data-[state=unsupported]/moq:inline-flex">
						{ locales.LiveStateUnsupported }
					</ui.Badge>
				</ui.CardAction>
			</ui.CardHeader>
			<ui.CardContent class="flex flex-col gap-4">
				<div class="aspect-video overflow-hidden rounded-lg bg-muted">
					// The page view's own stream: an idle publisher on the connection its watcher shares holds a name no
					// link watches, so it shadows nothing (#156).
					<moq-publish url={url} name={name} announce="source" class="block size-full">
						<video muted autoplay playsinline class="size-full object-contain"></video>
					</moq-publish>
				</div>
				// The link shows once the camera is on: on iOS Safari the broadcast flows while the page still reads
				// "connecting" (#156).
				<ui.Alert class="hidden group-data-[state=connecting]/moq:grid group-data-[state=live]/moq:grid group-data-[state=reconnecting]/moq:grid">
					<ui.AlertTitle>{ locales.LiveShare }</ui.AlertTitle>
					<ui.AlertDescription>
						<a
							href={share |> url}
							class="break-all underline underline-offset-4"
							dir="ltr"
							translate="no"
							data-testid="moq-share"
						>
							{ origin }{ share |> url }
						</a>
					</ui.AlertDescription>
				</ui.Alert>
				// DIAGNOSTICS on screen (#156): what the broadcaster's element reports, filled in by /static/moq.js.
				<p class="whitespace-pre-wrap break-all font-mono text-xs text-muted-foreground" dir="ltr" translate="no" data-moq-debug></p>
				<ui.Alert variant="destructive" class="hidden group-data-[error=denied]/moq:grid">
					<ui.AlertDescription>{ locales.LiveErrorDenied }</ui.AlertDescription>
				</ui.Alert>
				<ui.Alert variant="destructive" class="hidden group-data-[error=unavailable]/moq:grid">
					<ui.AlertDescription>{ locales.LiveErrorUnavailable }</ui.AlertDescription>
				</ui.Alert>
				<ui.Alert variant="destructive" class="hidden group-data-[error=failed]/moq:grid">
					<ui.AlertDescription>{ locales.LiveErrorFailed }</ui.AlertDescription>
				</ui.Alert>
			</ui.CardContent>
			<ui.CardFooter class="flex-wrap gap-2">
				<ui.Button
					size="sm"
					data-moq-action="start"
					class="group-data-[state=requesting]/moq:hidden group-data-[state=connecting]/moq:hidden group-data-[state=live]/moq:hidden group-data-[state=reconnecting]/moq:hidden"
				>
					{ locales.LiveStartCamera }
				</ui.Button>
				<ui.Button
					variant="outline"
					size="sm"
					data-moq-action="stop"
					class="hidden group-data-[state=requesting]/moq:inline-flex group-data-[state=connecting]/moq:inline-flex group-data-[state=live]/moq:inline-flex group-data-[state=reconnecting]/moq:inline-flex"
				>
					{ locales.LiveStop }
				</ui.Button>
				<ui.Button variant="outline" size="sm" data-moq-action="mute" class="group-data-[muted=true]/moq:hidden">
					{ locales.LiveMicOff }
				</ui.Button>
				<ui.Button
					variant="outline"
					size="sm"
					data-moq-action="unmute"
					class="hidden group-data-[muted=true]/moq:inline-flex"
				>
					{ locales.LiveMicOn }
				</ui.Button>
			</ui.CardFooter>
		</ui.Card>
	</section>
}
