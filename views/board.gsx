package views

import (
	"github.com/joeblew999/go-htmx4/ui"
	"github.com/joeblew999/go-htmx4/ui/icon"
)

// BoardPage is the shared board page, composed from gsxui components. With live (on Workers) it connects to
// /live/{topic} with hx-ws (loaded by Layout); the Room Durable Object pushes BoardFragment on every change and
// the online count into #presence (the label next to it is rendered here, in the page's language). Without it (`go run .`, no Durable Objects) only the poster's own response
// updates the board, and a badge says so.
component BoardPage(topic string, b Board, live bool) {
	<Layout title={M(ctx).BoardTitle()} path="/board">
		<div class="flex flex-col gap-2">
			<h1 class="text-3xl font-semibold tracking-tight">{ M(ctx).BoardTitle() }</h1>
			<p class="text-muted-foreground">{ M(ctx).BoardIntro() }</p>
		</div>
		<ui.Card>
			<ui.CardHeader>
				<ui.CardTitle>
					{ M(ctx).BoardTopic() }
					<code translate="no">{ topic }</code>
				</ui.CardTitle>
				<ui.CardDescription>{ M(ctx).BoardHint() }</ui.CardDescription>
				<ui.CardAction>
					{ if live {
						<ui.Badge variant="secondary">
							<icon.Users/>
							<span id="presence">…</span>
							<span>{ M(ctx).BoardOnline() }</span>
						</ui.Badge>
					} else {
						<ui.Badge variant="outline">
							<icon.WifiOff/>
							{ M(ctx).BoardNoLive() }
						</ui.Badge>
					} }
				</ui.CardAction>
			</ui.CardHeader>
			<ui.CardContent>
				{ if live {
					<div hx-ws:connect={"/live/" + topic + "?locale=" + LocaleKey(Loc(ctx).Data)} hx-swap="none">
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
						aria-label={M(ctx).BoardDecrement()}
						hx-post={URL(ctx, "/board/add?topic="+topic+"&delta=-1")}
						hx-swap="none"
					>
						<icon.Minus/>
					</ui.Button>
					<ui.Button
						type="button"
						variant="outline"
						size="icon"
						aria-label={M(ctx).BoardIncrement()}
						hx-post={URL(ctx, "/board/add?topic="+topic+"&delta=1")}
						hx-swap="none"
					>
						<icon.Plus/>
					</ui.Button>
				</ui.ButtonGroup>
				<form
					hx-post={URL(ctx, "/board/note?topic="+topic)}
					hx-swap="none"
					hx-on:htmx:after:request=js`this.reset()`
					class="flex min-w-0 flex-1 gap-2"
				>
					<ui.Input
						name="body"
						maxlength="280"
						placeholder={M(ctx).BoardNotePlaceholder()}
						autocomplete="off"
						aria-label={M(ctx).BoardNoteLabel()}
						required
					/>
					<ui.Button type="submit">
						<icon.Send/>
						{ M(ctx).BoardPost() }
					</ui.Button>
				</form>
			</ui.CardFooter>
		</ui.Card>
	</Layout>
}

// BoardFragment is the #board element as an out-of-band swap: the poster's response and the hx-ws push
// both use it. The Room and the page's version guard rely on its id="board", hx-swap-oob="true" and
// data-version (see TestBoardFragmentWireFormat).
component BoardFragment(b Board) {
	<section id="board" hx-swap-oob="true" data-version={b.Version} lang={Loc(ctx).Lang()} class="flex flex-col gap-6">
		<boardBody b={b}/>
	</section>
}

// boardSection is #board as the page renders it: no hx-swap-oob, or a boosted navigation to /board
// would treat the board as out-of-band content and drop it (no #board on the page it came from).
component boardSection(b Board) {
	<section id="board" data-version={b.Version} lang={Loc(ctx).Lang()} class="flex flex-col gap-6">
		<boardBody b={b}/>
	</section>
}

component boardBody(b Board) {
	<div class="flex items-baseline gap-3">
		<output class="text-6xl font-semibold tracking-tight tabular-nums">{ Num(ctx, b.Value) }</output>
		<span class="text-sm text-muted-foreground">{ M(ctx).BoardVersion(float64(b.Version)) }</span>
	</div>
	{ if len(b.Notes) == 0 {
		<ui.Empty>
			<ui.EmptyHeader>
				<ui.EmptyMedia variant="icon">
					<icon.MessageSquare/>
				</ui.EmptyMedia>
				<ui.EmptyTitle>{ M(ctx).BoardEmptyTitle() }</ui.EmptyTitle>
				<ui.EmptyDescription>{ M(ctx).BoardEmptyDescription() }</ui.EmptyDescription>
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
							<time datetime={NoteISO(n.CreatedAt)} data-relative-time>{ RelativeSince(ctx, n.CreatedAt) }</time>
						</ui.ItemDescription>
					</ui.ItemContent>
				</ui.Item>
			} }
		</ui.ItemGroup>
	} }
}
