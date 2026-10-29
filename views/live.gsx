package views

import (
	"github.com/joeblew999/go-htmx4/locales"
	"github.com/joeblew999/go-htmx4/ui"
)

// LivePage is live video over MoQ (#156): a broadcaster's camera and a watcher, through a MoQ relay. The media is
// @moq's elements (<moq-publish>, <moq-watch>, vendored into /static/moq.js); everything around them is gsxui.
//
// Every label for every state is rendered here, translated; /static/moq.js only sets the section's data-state (and
// data-paused, data-muted), and Tailwind's group-data variants show the one that applies. The script is the page's
// own: the nav reaches this page with a full load, not a boosted swap.
component LivePage(p LiveProps) {
	<Layout title={locales.LiveTitle} description={locales.LiveDescription}>
		<h1 class="text-3xl font-semibold tracking-tight">{ locales.LiveHeading }</h1>
		<p class="text-muted-foreground">{ locales.LiveIntro }</p>
		<LiveBroadcast url={p.PublishURL} name={p.Name}/>
		<LiveWatch url={p.WatchURL} name={p.Name}/>
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
					<moq-watch url={url} name={name} muted class="block size-full">
						<canvas class="size-full object-contain" aria-label={locales.LiveWatchTitle |> text}></canvas>
					</moq-watch>
				</div>
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
component LiveBroadcast(url string, name string) {
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
					<moq-publish url={url} name={name} announce="source" class="block size-full">
						<video muted autoplay playsinline class="size-full object-contain"></video>
					</moq-publish>
				</div>
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
