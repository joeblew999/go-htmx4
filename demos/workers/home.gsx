package main

import (
	"github.com/joeblew999/go-htmx4/demos/workers/ui"
	"github.com/joeblew999/go-htmx4/demos/workers/ui/icon"
)

// HomePage shows htmx 4 round-trips against Go on Workers, composed from gsxui components.
component HomePage(target string, env string) {
	<Layout title="Home" path="/" htmxConfig="" head={ <></> }>
		<div class="flex flex-col gap-2">
			<h1 class="text-3xl font-semibold tracking-tight">htmx 4 on Cloudflare Workers</h1>
			<p class="text-muted-foreground">Go compiled with TinyGo and served by workers-go. No Node, no bundler.</p>
			<div class="flex flex-wrap gap-2">
				<ui.Badge variant="secondary">Served by <span id="target">{ target }</span></ui.Badge>
				<ui.Badge variant="outline">env <span id="env">{ env }</span></ui.Badge>
			</div>
		</div>
		<ui.Card>
			<ui.CardHeader>
				<ui.CardTitle>Fragment round-trip</ui.CardTitle>
				<ui.CardDescription>hx-get a fragment rendered by Go on Workers.</ui.CardDescription>
			</ui.CardHeader>
			<ui.CardContent class="flex flex-wrap items-center gap-3">
				<ui.Button type="button" hx-get="/fragments/now" hx-target="#now" hx-swap="innerHTML">
					<icon.Zap/> Ask the server
				</ui.Button>
				<output id="now" class="text-sm text-muted-foreground">not asked yet</output>
			</ui.CardContent>
		</ui.Card>
		<ui.Card>
			<ui.CardHeader>
				<ui.CardTitle>Form post</ui.CardTitle>
				<ui.CardDescription>hx-post a form; the escaped greeting comes back as a fragment.</ui.CardDescription>
			</ui.CardHeader>
			<ui.CardContent class="flex flex-col gap-3">
				<form hx-post="/greet" hx-target="#greeting" hx-swap="innerHTML">
					<ui.Field>
						<ui.FieldLabel for="name">Name</ui.FieldLabel>
						<div class="flex gap-2">
							<ui.Input id="name" name="name" placeholder="Jamie Lee" autocomplete="off"/>
							<ui.Button type="submit">
								<icon.Send/> Greet
							</ui.Button>
						</div>
					</ui.Field>
				</form>
				<div id="greeting" class="text-sm"></div>
			</ui.CardContent>
		</ui.Card>
		<ui.Card>
			<ui.CardHeader>
				<ui.CardTitle>No memory between requests</ui.CardTitle>
				<ui.CardDescription>
					The counter is a package-level Go variable. On Workers every request starts a fresh Go runtime, so it stays at
					1. Under go run . it keeps counting.
				</ui.CardDescription>
			</ui.CardHeader>
			<ui.CardContent class="flex items-center gap-3">
				<ui.Button type="button" variant="outline" hx-post="/count" hx-target="#count" hx-swap="innerHTML">
					<icon.Plus/> Count
				</ui.Button>
				<output id="count" class="text-sm text-muted-foreground">0</output>
			</ui.CardContent>
		</ui.Card>
		<ui.Card>
			<ui.CardHeader>
				<ui.CardTitle>Live updates</ui.CardTitle>
				<ui.CardDescription>
					D1 plus a Durable Object per topic push every change to all open tabs over hx-ws.
				</ui.CardDescription>
			</ui.CardHeader>
			<ui.CardContent>
				<ui.Button href="/board">
					<icon.Radio/> Open the shared board
				</ui.Button>
			</ui.CardContent>
		</ui.Card>
	</Layout>
}
