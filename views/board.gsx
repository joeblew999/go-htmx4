package views

import (
	"github.com/joeblew999/go-htmx4/intl"
	"github.com/joeblew999/go-htmx4/locales"
	"github.com/joeblew999/go-htmx4/ui"
	"github.com/joeblew999/go-htmx4/ui/icon"
)

// BoardPage is the shared board page, composed from gsxui components. With live (on Workers) it connects to
// /live/{topic} with hx-ws (loaded by Layout); the Room Durable Object pushes BoardFragment on every change and
// the online count into #presence (the label next to it is rendered here, in the page's language). Without it (go test, no Durable Objects) only the poster's own response
// updates the board, and a badge says so.
component BoardPage(topic string, b BoardState, live bool) {
	<Layout title={locales.BoardTitle} description={locales.BoardIntro}>
		<div class="flex flex-col gap-2">
			<h1 class="text-3xl font-semibold tracking-tight">{ locales.BoardTitle }</h1>
			<p class="text-muted-foreground">{ locales.BoardIntro }</p>
		</div>
		<ui.Card>
			<ui.CardHeader>
				<ui.CardTitle>
					{ locales.BoardTopic }
					<code translate="no">{ topic }</code>
				</ui.CardTitle>
				<ui.CardDescription>{ locales.BoardHint }</ui.CardDescription>
				<ui.CardAction>
					{ if live {
						<ui.Badge variant="secondary">
							<icon.Users/>
							<span id="presence">…</span>
							<span>{ locales.BoardOnline }</span>
						</ui.Badge>
					} else {
						<ui.Badge variant="outline">
							<icon.WifiOff/>
							{ locales.BoardNoLive }
						</ui.Badge>
					} }
				</ui.CardAction>
			</ui.CardHeader>
			<ui.CardContent>
				{ if live {
					<div hx-ws:connect=f`/live/@{topic}?locale=@{intl.Key(intl.Current(ctx))}` hx-swap="none">
						<boardSection b={b}/>
					</div>
				} else {
					<boardSection b={b}/>
				} }
			</ui.CardContent>
			<ui.CardFooter class="flex flex-wrap items-center gap-3">
				<ui.ButtonGroup>
					<ui.Button
						type="button"
						variant="outline"
						size="icon"
						aria-label={locales.BoardDecrement}
						hx-post={f`/board/add?topic=@{topic}&delta=-1` |> localize}
						hx-swap="none"
					>
						<icon.Minus/>
					</ui.Button>
					<ui.Button
						type="button"
						variant="outline"
						size="icon"
						aria-label={locales.BoardIncrement}
						hx-post={f`/board/add?topic=@{topic}&delta=1` |> localize}
						hx-swap="none"
					>
						<icon.Plus/>
					</ui.Button>
				</ui.ButtonGroup>
				<form
					hx-post={f`/board/note?topic=@{topic}` |> localize}
					hx-swap="none"
					hx-on:htmx:after:request=js`this.reset()`
					class="flex min-w-0 flex-1 gap-2"
				>
					<ui.Input
						name="body"
						maxlength="280"
						placeholder={locales.BoardNotePlaceholder}
						autocomplete="off"
						aria-label={locales.BoardNoteLabel}
						required
					/>
					<ui.Button type="submit">
						<icon.Send/>
						{ locales.BoardPost }
					</ui.Button>
				</form>
			</ui.CardFooter>
		</ui.Card>
		<section class="flex flex-col gap-3">
			<h2 class="text-xl font-semibold tracking-tight">{ locales.BoardHowTitle }</h2>
			<ol class="flex list-decimal flex-col gap-2 ps-5 text-muted-foreground">
				<li>{ locales.BoardHowWrite }</li>
				<li>{ locales.BoardHowPush }</li>
				<li>{ locales.BoardHowSockets }</li>
				<li>{ locales.BoardHowOrder }</li>
				<li>{ locales.BoardHowLimits }</li>
			</ol>
		</section>
	</Layout>
}

// BoardFragment is the #board element as an out-of-band swap: the poster's response and the hx-ws push
// both use it. The Room and the page's version guard rely on its id="board", hx-swap-oob="true" and
// data-version (see TestBoardFragmentWireFormat).
component BoardFragment(b BoardState) {
	<boardSection b={b} oob={true}/>
}

// boardSection is #board; as the page renders it (oob false) it has no hx-swap-oob, or a boosted navigation to /board
// would treat the board as out-of-band content and drop it (no #board on the page it came from).
component boardSection(b BoardState, oob bool) {
	<section
		id="board"
		{ if oob {
			hx-swap-oob="true"
		} }
		data-version={b.Version}
		lang={intl.Current(ctx).ID()}
		class="flex flex-col gap-6"
	>
		<boardBody b={b}/>
	</section>
}

component boardBody(b BoardState) {
	<div class="flex items-baseline gap-3">
		<output class="text-6xl font-semibold tracking-tight tabular-nums">{ b.Value |> number }</output>
		<span class="text-sm text-muted-foreground">{ locales.BoardVersion(float64(b.Version)) }</span>
	</div>
	{ if len(b.Notes) == 0 {
		<ui.Empty>
			<ui.EmptyHeader>
				<ui.EmptyMedia variant="icon">
					<icon.MessageSquare/>
				</ui.EmptyMedia>
				<ui.EmptyTitle>{ locales.BoardEmptyTitle }</ui.EmptyTitle>
				<ui.EmptyDescription>{ locales.BoardEmptyDescription }</ui.EmptyDescription>
			</ui.EmptyHeader>
		</ui.Empty>
	} else {
		<ui.ItemGroup>
			{ for i, n := range b.Notes {
				{ if i > 0 {
					<ui.ItemSeparator/>
				} }
				<ui.Item size="sm">
					<ui.ItemMedia variant="icon">
						<icon.MessageSquare/>
					</ui.ItemMedia>
					<ui.ItemContent>
						<ui.ItemTitle dir="auto">{ n.Body }</ui.ItemTitle>
						<ui.ItemDescription>
							{ n.Created() |> since }
						</ui.ItemDescription>
					</ui.ItemContent>
				</ui.Item>
			} }
		</ui.ItemGroup>
	} }
}
