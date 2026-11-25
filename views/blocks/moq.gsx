package blocks

import (
	"github.com/gsxhq/gsx"
	"github.com/joeblew999/go-htmx4/locales"
	"github.com/joeblew999/go-htmx4/ui"
	"github.com/joeblew999/go-htmx4/ui/icon"
)

// Live video over Media over QUIC (MoQ): @moq's <moq-watch> and <moq-publish> elements, in gsxui cards. The elements
// and the glue between them and this markup are /blocks/moq.js (the blocks' js/moq, bundled from npm with no Node).
//
// Every label for every state is rendered here, translated; moq.js only sets the section's data-state (and
// data-paused, data-muted, data-error, data-copied, data-can-share), and Tailwind's group-data variants show the one
// that applies. Each card's connection details are machine values the script writes. moq.js is written on gsxui's
// core: the page loads gsxui's behaviours from /gsxui/ (gsxui.json's js, served there). Each card's id (from its
// broadcast's name) keeps a boosted navigation's morph from turning it into another element.
//
// The relay: Cloudflare's open one is https://draft-14.cloudflare.mediaoverquic.com/ (moq-transport draft-14, no
// token). @moq is held at its last releases that speak drafts 14 and 16, which Cloudflare's relays speak; when they
// ship draft-18, blocks moves @moq on and drops its patches (joeblew999/go-htmx4#158). The page's Content Security
// Policy must allow the relay in connect-src, and blob: in script-src and worker-src (the elements' audio worklets
// and codec workers).
//
// From go tool blocks add moq: the app's own now, to change as it likes.

// MoqWatch plays the broadcast called name on the relay at url. It starts muted: browsers only play sound after a
// click.
component MoqWatch(url string, name string, attrs gsx.Attrs) {
	<section
		class="group/moq"
		data-moq="watch"
		data-state="connecting"
		data-paused="false"
		data-muted="true"
		id=f`moq-watch-@{name}`
		{ attrs... }
	>
		<ui.Card>
			<ui.CardHeader>
				<ui.CardTitle>{ locales.BlocksMoqWatchTitle }</ui.CardTitle>
				<ui.CardDescription>{ locales.BlocksMoqWatchDescription }</ui.CardDescription>
				<ui.CardAction role="status" aria-live="polite">
					<ui.Badge variant="secondary" class="hidden group-data-[state=connecting]/moq:inline-flex">
						{ locales.BlocksMoqStateConnecting }
					</ui.Badge>
					<ui.Badge variant="secondary" class="hidden group-data-[state=waiting]/moq:inline-flex">
						{ locales.BlocksMoqStateWaiting }
					</ui.Badge>
					<ui.Badge class="hidden group-data-[state=playing]/moq:inline-flex">
						{ locales.BlocksMoqStatePlaying }
					</ui.Badge>
					<ui.Badge variant="secondary" class="hidden group-data-[state=paused]/moq:inline-flex">
						{ locales.BlocksMoqStatePaused }
					</ui.Badge>
					<ui.Badge variant="secondary" class="hidden group-data-[state=reconnecting]/moq:inline-flex">
						{ locales.BlocksMoqStateReconnecting }
					</ui.Badge>
					<ui.Badge variant="outline" class="hidden group-data-[state=ended]/moq:inline-flex">
						{ locales.BlocksMoqStateEnded }
					</ui.Badge>
					<ui.Badge variant="destructive" class="hidden group-data-[state=unsupported]/moq:inline-flex">
						{ locales.BlocksMoqStateUnsupported }
					</ui.Badge>
				</ui.CardAction>
			</ui.CardHeader>
			<ui.CardContent class="flex flex-col gap-4">
				<div class="aspect-video overflow-hidden rounded-lg bg-muted" data-moq-frame>
					// announced="false": subscribe by name. Cloudflare's draft-14 relays don't answer namespace subscriptions, so a
					// watcher waiting for the broadcast to be announced would wait forever.
					<moq-watch url={url} name={name} muted announced="false" class="block size-full">
						<canvas class="size-full object-contain" aria-label={locales.BlocksMoqWatchTitle |> text}></canvas>
					</moq-watch>
				</div>
				<moqDetails/>
			</ui.CardContent>
			<ui.CardFooter class="flex-wrap gap-2">
				<ui.Button variant="outline" size="sm" data-moq-action="pause" class="group-data-[paused=true]/moq:hidden">
					{ locales.BlocksMoqPause }
				</ui.Button>
				<ui.Button
					variant="outline"
					size="sm"
					data-moq-action="play"
					class="hidden group-data-[paused=true]/moq:inline-flex"
				>
					{ locales.BlocksMoqPlay }
				</ui.Button>
				<ui.Button variant="outline" size="sm" data-moq-action="unmute" class="group-data-[muted=false]/moq:hidden">
					{ locales.BlocksMoqUnmute }
				</ui.Button>
				<ui.Button
					variant="outline"
					size="sm"
					data-moq-action="mute"
					class="hidden group-data-[muted=false]/moq:inline-flex"
				>
					{ locales.BlocksMoqMute }
				</ui.Button>
				<ui.Button variant="outline" size="sm" data-moq-action="fullscreen">{ locales.BlocksMoqFullscreen }</ui.Button>
			</ui.CardFooter>
		</ui.Card>
		<script type="module" src="/blocks/moq.js"></script>
	</section>
}

// MoqBroadcast sends the reader's camera and microphone to the relay at url as the broadcast called name. Nothing is
// captured until they press start. On air, it offers link, the address that watches it, to copy or to hand to the
// system's share sheet.
component MoqBroadcast(url string, name string, link string, attrs gsx.Attrs) {
	<section
		class="group/moq"
		data-moq="broadcast"
		data-state="idle"
		data-muted="false"
		data-copied="false"
		data-can-share="false"
		id=f`moq-broadcast-@{name}`
		{ attrs... }
	>
		<ui.Card>
			<ui.CardHeader>
				<ui.CardTitle>{ locales.BlocksMoqBroadcastTitle }</ui.CardTitle>
				<ui.CardDescription>{ locales.BlocksMoqBroadcastDescription }</ui.CardDescription>
				<ui.CardAction role="status" aria-live="polite">
					<ui.Badge variant="outline" class="hidden group-data-[state=idle]/moq:inline-flex">
						{ locales.BlocksMoqStateIdle }
					</ui.Badge>
					<ui.Badge variant="secondary" class="hidden group-data-[state=requesting]/moq:inline-flex">
						{ locales.BlocksMoqStateRequesting }
					</ui.Badge>
					<ui.Badge variant="secondary" class="hidden group-data-[state=connecting]/moq:inline-flex">
						{ locales.BlocksMoqStateConnecting }
					</ui.Badge>
					<ui.Badge variant="destructive" class="hidden group-data-[state=live]/moq:inline-flex">
						{ locales.BlocksMoqStateOnAir }
					</ui.Badge>
					<ui.Badge variant="secondary" class="hidden group-data-[state=reconnecting]/moq:inline-flex">
						{ locales.BlocksMoqStateReconnecting }
					</ui.Badge>
					<ui.Badge variant="destructive" class="hidden group-data-[state=error]/moq:inline-flex">
						{ locales.BlocksMoqStateFailed }
					</ui.Badge>
					<ui.Badge variant="destructive" class="hidden group-data-[state=unsupported]/moq:inline-flex">
						{ locales.BlocksMoqStateUnsupported }
					</ui.Badge>
				</ui.CardAction>
			</ui.CardHeader>
			<ui.CardContent class="flex flex-col gap-4">
				<div class="relative aspect-video overflow-hidden rounded-lg bg-muted">
					// The camera's place, before it starts.
					<div
						class="absolute inset-0 hidden items-center justify-center text-muted-foreground group-data-[state=idle]/moq:flex"
						aria-hidden="true"
					>
						<icon.VideoOff class="size-10"/>
					</div>
					<moq-publish url={url} name={name} announce="source" class="block size-full">
						<video muted autoplay playsinline class="size-full object-contain"></video>
					</moq-publish>
				</div>
				// On air: the link to send, to copy or hand to the system's share sheet (where the browser has one).
				<ui.Field class="hidden group-data-[state=live]/moq:flex group-data-[state=reconnecting]/moq:flex">
					<ui.FieldLabel for=f`moq-link-@{name}`>{ locales.BlocksMoqShare }</ui.FieldLabel>
					<ui.InputGroup>
						<ui.InputGroupInput
							id=f`moq-link-@{name}`
							value={link}
							readonly
							dir="ltr"
							translate="no"
							data-moq-link
						/>
						<ui.InputGroupAddon align="inline-end">
							<ui.InputGroupButton data-moq-action="copy">
								<icon.Copy class="group-data-[copied=true]/moq:hidden"/>
								<icon.Check class="hidden group-data-[copied=true]/moq:block"/>
								// On a phone the buttons are their icons, so the link keeps the room; the words stay for screen readers.
								<span class="sr-only sm:not-sr-only group-data-[copied=true]/moq:hidden">{ locales.BlocksMoqCopyLink }</span>
								<span class="sr-only sm:not-sr-only hidden group-data-[copied=true]/moq:inline">{ locales.BlocksMoqCopied }</span>
							</ui.InputGroupButton>
							<ui.InputGroupButton data-moq-action="share" class="hidden group-data-[can-share=true]/moq:inline-flex">
								<icon.Share/>
								<span class="sr-only sm:not-sr-only">{ locales.BlocksMoqShareLink }</span>
							</ui.InputGroupButton>
						</ui.InputGroupAddon>
					</ui.InputGroup>
					<ui.FieldDescription>{ locales.BlocksMoqKeepOpen }</ui.FieldDescription>
				</ui.Field>
				<ui.Alert variant="destructive" class="hidden group-data-[error=denied]/moq:grid">
					<ui.AlertDescription>{ locales.BlocksMoqErrorDenied }</ui.AlertDescription>
				</ui.Alert>
				<ui.Alert variant="destructive" class="hidden group-data-[error=unavailable]/moq:grid">
					<ui.AlertDescription>{ locales.BlocksMoqErrorUnavailable }</ui.AlertDescription>
				</ui.Alert>
				<ui.Alert variant="destructive" class="hidden group-data-[error=failed]/moq:grid">
					<ui.AlertDescription>{ locales.BlocksMoqErrorFailed }</ui.AlertDescription>
				</ui.Alert>
				<moqDetails/>
			</ui.CardContent>
			<ui.CardFooter class="flex-wrap gap-2">
				<ui.Button
					size="sm"
					data-moq-action="start"
					class="group-data-[state=requesting]/moq:hidden group-data-[state=connecting]/moq:hidden group-data-[state=live]/moq:hidden group-data-[state=reconnecting]/moq:hidden"
				>
					{ locales.BlocksMoqStartCamera }
				</ui.Button>
				<ui.Button
					variant="outline"
					size="sm"
					data-moq-action="stop"
					class="hidden group-data-[state=requesting]/moq:inline-flex group-data-[state=connecting]/moq:inline-flex group-data-[state=live]/moq:inline-flex group-data-[state=reconnecting]/moq:inline-flex"
				>
					{ locales.BlocksMoqStop }
				</ui.Button>
				<ui.Button variant="outline" size="sm" data-moq-action="mute" class="group-data-[muted=true]/moq:hidden">
					{ locales.BlocksMoqMicOff }
				</ui.Button>
				<ui.Button
					variant="outline"
					size="sm"
					data-moq-action="unmute"
					class="hidden group-data-[muted=true]/moq:inline-flex"
				>
					{ locales.BlocksMoqMicOn }
				</ui.Button>
			</ui.CardFooter>
		</ui.Card>
		<script type="module" src="/blocks/moq.js"></script>
	</section>
}

// moqDetails is a card's connection details, closed until opened: the @moq versions, the relay, the connection and
// its transport, the browser's features and @moq's last warning, for testing on phones. moq.js writes them while it's
// open; they are machine values, not text to translate.
component moqDetails() {
	<ui.Collapsible class="flex flex-col gap-2">
		<ui.CollapsibleTrigger
			class="flex cursor-pointer items-center justify-between gap-4 rounded-md border px-3 py-2 text-sm font-medium"
		>
			<span>{ locales.BlocksMoqDetails }</span>
			<span aria-hidden="true">
				<icon.ChevronsUpDown class="size-4"/>
			</span>
		</ui.CollapsibleTrigger>
		<ui.CollapsibleContent>
			<pre
				class="whitespace-pre-wrap wrap-break-word rounded-md bg-muted p-3 font-mono text-xs text-muted-foreground"
				dir="ltr"
				translate="no"
				data-moq-debug
			></pre>
		</ui.CollapsibleContent>
	</ui.Collapsible>
}
