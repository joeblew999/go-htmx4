package views

import "github.com/joeblew999/go-htmx4/demos/gsxui/ui"

// Home shows gsxui components driven by htmx 4: server round-trips that return
// gsx fragments, an out-of-band toast, lazily loaded dialog and tab content, and
// client-side state with hx-live.
component Home(flavours []string, components []string) {
	<Layout title="Home" path="/">
		<div class="flex flex-col gap-2">
			<h1 class="text-3xl font-semibold tracking-tight">gsxui + htmx 4</h1>
			<p class="text-muted-foreground">
				Server-rendered gsx components, htmx 4 for requests, hx-live for local state. No Node, no bundler.
			</p>
		</div>
		<GreetCard flavours={flavours}/>
		<div class="grid gap-6 md:grid-cols-2">
			<DialogCard/>
			<TabsCard/>
		</div>
		<LiveCard components={components}/>
	</Layout>
}

component GreetCard(flavours []string) {
	<ui.Card>
		<ui.CardHeader>
			<ui.CardTitle>Server round-trip</ui.CardTitle>
			<ui.CardDescription>
				hx-post a gsxui form. The server answers with a gsx fragment and an out-of-band toast.
			</ui.CardDescription>
		</ui.CardHeader>
		<ui.CardContent class="flex flex-col gap-4">
			<form hx-post="/greet" hx-target="#greeting" class="flex flex-col gap-4">
				<ui.FieldGroup>
					<ui.Field>
						<ui.FieldLabel for="name">Name</ui.FieldLabel>
						<ui.Input id="name" name="name" placeholder="Jamie Lee" required/>
					</ui.Field>
					<ui.Field>
						<ui.FieldLabel for="flavour">Favourite part of the stack</ui.FieldLabel>
						<ui.NativeSelect id="flavour" name="flavour">
							{ for _, f := range flavours {
								<ui.NativeSelectOption value={f}>{ f }</ui.NativeSelectOption>
							} }
						</ui.NativeSelect>
					</ui.Field>
					<ui.Field orientation="horizontal">
						<ui.Switch id="shout" name="shout" value="on"/>
						<ui.FieldLabel for="shout">Shout it</ui.FieldLabel>
					</ui.Field>
				</ui.FieldGroup>
				<div>
					<ui.Button type="submit">Say hello</ui.Button>
				</div>
			</form>
			<div id="greeting"></div>
		</ui.CardContent>
	</ui.Card>
}

component DialogCard() {
	<ui.Card>
		<ui.CardHeader>
			<ui.CardTitle>Dialog + hx-get</ui.CardTitle>
			<ui.CardDescription>The dialog body is fetched from the server each time it opens.</ui.CardDescription>
		</ui.CardHeader>
		<ui.CardContent>
			<ui.Dialog>
				<ui.Button
					variant="outline"
					data-gsxui-slot-dialog-trigger
					aria-haspopup="dialog"
					aria-expanded="false"
					hx-get="/fragments/server-info"
					hx-target="#server-info"
				>
					Show server info
				</ui.Button>
				<ui.DialogContent>
					<ui.DialogHeader>
						<ui.DialogTitle>Server info</ui.DialogTitle>
						<ui.DialogDescription>Rendered by a gsx component on request.</ui.DialogDescription>
					</ui.DialogHeader>
					<div id="server-info" class="text-sm text-muted-foreground">Loading…</div>
					<ui.DialogFooter showCloseButton={true}></ui.DialogFooter>
				</ui.DialogContent>
			</ui.Dialog>
		</ui.CardContent>
	</ui.Card>
}

component TabsCard() {
	<ui.Card>
		<ui.CardHeader>
			<ui.CardTitle>Tabs + lazy panel</ui.CardTitle>
			<ui.CardDescription>The Stats panel loads once, the first time its tab is clicked.</ui.CardDescription>
		</ui.CardHeader>
		<ui.CardContent>
			<ui.Tabs value="overview">
				<ui.TabsList>
					<ui.TabsTrigger value="overview" selected>Overview</ui.TabsTrigger>
					<ui.TabsTrigger value="stats" hx-get="/fragments/stats" hx-target="#tab-stats" hx-trigger="click once">
						Stats
					</ui.TabsTrigger>
				</ui.TabsList>
				<ui.TabsContent value="overview" selected class="pt-3 text-sm">
					This panel was rendered with the page.
				</ui.TabsContent>
				<ui.TabsContent value="stats" class="pt-3 text-sm">
					<div id="tab-stats" class="text-muted-foreground">Loading…</div>
				</ui.TabsContent>
			</ui.Tabs>
		</ui.CardContent>
	</ui.Card>
}

// LiveCard uses the htmx 4 hx-live extension: state lives in the DOM
// (data-count), :text re-renders after mutations, and q() queries relative to
// the element (patterns from jackielii/hx-live-vs-alpine).
component LiveCard(components []string) {
	<ui.Card>
		<ui.CardHeader>
			<ui.CardTitle>Client state with hx-live</ui.CardTitle>
			<ui.CardDescription>No server round-trip: a counter and a live filter.</ui.CardDescription>
		</ui.CardHeader>
		<ui.CardContent class="flex flex-col gap-4">
			<div data-count="0" class="flex items-center gap-3">
				<ui.Button variant="outline" hx-on:click=js`data.count++`>Increment</ui.Button>
				<span class="text-sm">
					Count:
					<ui.Badge variant="secondary" :text=js`data.count`>0</ui.Badge>
				</span>
			</div>
			<div class="flex flex-col gap-2">
				<ui.Input placeholder="Filter gsxui components…" aria-label="Filter components"/>
				<ul
					class="flex flex-wrap gap-2"
					hx-live=js`for (let li of q('li in this')) li.hidden = !li.textContent.toLowerCase().includes(q('previous input').value.toLowerCase())`
				>
					{ for _, c := range components {
						<li>
							<ui.Badge variant="outline">{ c }</ui.Badge>
						</li>
					} }
				</ul>
			</div>
		</ui.CardContent>
	</ui.Card>
}
