package views

import (
	"github.com/joeblew999/go-htmx4/ui"
	"github.com/joeblew999/go-htmx4/ui/icon"
)

// BoardPage is the shared board page, composed from gsxui components. It connects to /live/{topic} with hx-ws
// (loaded by Layout); the Room Durable Object pushes BoardFragment on every change and "N online" into
// #presence.
component BoardPage(topic string, b Board) {
	<Layout title="Shared board" path="/board">
		<div class="flex flex-col gap-2">
			<h1 class="text-3xl font-semibold tracking-tight">Shared board</h1>
			<p class="text-muted-foreground">
				Go writes to D1, then the topic's Durable Object pushes the fragment to every open tab over hx-ws.
			</p>
		</div>
		<ui.Card>
			<ui.CardHeader>
				<ui.CardTitle>Topic <code>{ topic }</code></ui.CardTitle>
				<ui.CardDescription>Open this page in another tab and change something.</ui.CardDescription>
				<ui.CardAction>
					<ui.Badge variant="secondary">
						<icon.Users/>
						<span id="presence">connecting…</span>
					</ui.Badge>
				</ui.CardAction>
			</ui.CardHeader>
			<ui.CardContent>
				<div hx-ws:connect={"/live/" + topic} hx-swap="none">
					<boardSection b={b}/>
				</div>
			</ui.CardContent>
			<ui.CardFooter class="flex flex-wrap items-center gap-3">
				<ui.ButtonGroup>
					<ui.Button
						type="button"
						variant="outline"
						size="icon"
						aria-label="Decrement"
						hx-post={"/board/add?topic=" + topic + "&delta=-1"}
						hx-swap="none"
					>
						<icon.Minus/>
					</ui.Button>
					<ui.Button
						type="button"
						variant="outline"
						size="icon"
						aria-label="Increment"
						hx-post={"/board/add?topic=" + topic + "&delta=1"}
						hx-swap="none"
					>
						<icon.Plus/>
					</ui.Button>
				</ui.ButtonGroup>
				<form
					hx-post={"/board/note?topic=" + topic}
					hx-swap="none"
					hx-on:htmx:after:request=js`this.reset()`
					class="flex min-w-0 flex-1 gap-2"
				>
					<ui.Input
						name="body"
						maxlength="280"
						placeholder="Leave a note"
						autocomplete="off"
						aria-label="Note"
						required
					/>
					<ui.Button type="submit">
						<icon.Send/> Post
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
	<section id="board" hx-swap-oob="true" data-version={b.Version} class="flex flex-col gap-6">
		<boardBody b={b}/>
	</section>
}

// boardSection is #board as the page renders it: no hx-swap-oob, or a boosted navigation to /board
// would treat the board as out-of-band content and drop it (no #board on the page it came from).
component boardSection(b Board) {
	<section id="board" data-version={b.Version} class="flex flex-col gap-6">
		<boardBody b={b}/>
	</section>
}

component boardBody(b Board) {
	<div class="flex items-baseline gap-3">
		<output class="text-6xl font-semibold tracking-tight tabular-nums">{ b.Value }</output>
		<span class="text-sm text-muted-foreground">version { b.Version }</span>
	</div>
	{ if len(b.Notes) == 0 {
		<ui.Empty>
			<ui.EmptyHeader>
				<ui.EmptyMedia variant="icon">
					<icon.MessageSquare/>
				</ui.EmptyMedia>
				<ui.EmptyTitle>No notes yet</ui.EmptyTitle>
				<ui.EmptyDescription>Leave one below. Every open tab sees it instantly.</ui.EmptyDescription>
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
						<ui.ItemTitle>{ n.Body }</ui.ItemTitle>
						<ui.ItemDescription>{ n.CreatedAt }</ui.ItemDescription>
					</ui.ItemContent>
				</ui.Item>
			} }
		</ui.ItemGroup>
	} }
}
