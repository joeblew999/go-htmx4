package views

import (
	"github.com/joeblew999/go-htmx4/ui"
	"github.com/joeblew999/go-htmx4/ui/icon"
)

// HomePage shows gsxui components driven by htmx 4: server round-trips that return gsx fragments, an
// out-of-band toast, lazily loaded dialog and tab content, client-side state with hx-live, and the live
// board.
component HomePage(target string, env string, flavours []string, components []string) {
	<Layout title={M(ctx).HomeTitle()} path="/">
		<div class="flex flex-col gap-2">
			<h1 class="text-3xl font-semibold tracking-tight">{ M(ctx).HomeHeading() }</h1>
			<p class="text-muted-foreground">{ M(ctx).HomeIntro() }</p>
			<div class="flex flex-wrap gap-2">
				<ui.Badge variant="secondary">
					{ M(ctx).HomeServedBy() }
					<span id="target" translate="no">{ target }</span>
				</ui.Badge>
				<ui.Badge variant="outline">
					{ M(ctx).HomeEnv() }
					<span id="env" translate="no">{ env }</span>
				</ui.Badge>
			</div>
		</div>
		<GreetCard flavours={flavours}/>
		<div class="grid gap-6 md:grid-cols-2">
			<DialogCard/>
			<TabsCard/>
		</div>
		<LiveCard components={components}/>
		<BoardCard/>
	</Layout>
}

component GreetCard(flavours []string) {
	<ui.Card>
		<ui.CardHeader>
			<ui.CardTitle>{ M(ctx).HomeGreetTitle() }</ui.CardTitle>
			<ui.CardDescription>{ M(ctx).HomeGreetDescription() }</ui.CardDescription>
		</ui.CardHeader>
		<ui.CardContent class="flex flex-col gap-4">
			<form hx-post={URL(ctx, "/greet")} hx-target="#greeting" class="flex flex-col gap-4">
				<ui.FieldGroup>
					<ui.Field>
						<ui.FieldLabel for="name">{ M(ctx).HomeGreetName() }</ui.FieldLabel>
						<ui.Input id="name" name="name" placeholder={M(ctx).HomeGreetNamePlaceholder()} required/>
					</ui.Field>
					<ui.Field>
						<ui.FieldLabel for="flavour">{ M(ctx).HomeGreetFlavour() }</ui.FieldLabel>
						<ui.NativeSelect id="flavour" name="flavour" translate="no">
							{ for _, f := range flavours {
								<ui.NativeSelectOption value={f}>{ f }</ui.NativeSelectOption>
							} }
						</ui.NativeSelect>
					</ui.Field>
					<ui.Field orientation="horizontal">
						<ui.Switch id="shout" name="shout" value="on"/>
						<ui.FieldLabel for="shout">{ M(ctx).HomeGreetShout() }</ui.FieldLabel>
					</ui.Field>
				</ui.FieldGroup>
				<div class="flex gap-2">
					<ui.Button type="submit">
						<icon.Send/>
						{ M(ctx).HomeGreetSubmit() }
					</ui.Button>
					<ui.Button
						variant="ghost"
						type="button"
						hx-action={URL(ctx, "/greet")}
						hx-method="delete"
						hx-target="#greeting"
					>
						{ M(ctx).HomeGreetClear() }
					</ui.Button>
				</div>
			</form>
			<div id="greeting"></div>
		</ui.CardContent>
	</ui.Card>
}

// DialogCard's close button is gsxui DialogFooter's own markup (ui/dialog.gsx) with a translated label: gsxui
// hard-codes "Close" in DialogContent and DialogFooter, so both built-in buttons are turned off.
component DialogCard() {
	<ui.Card>
		<ui.CardHeader>
			<ui.CardTitle>{ M(ctx).HomeDialogTitle() }</ui.CardTitle>
			<ui.CardDescription>{ M(ctx).HomeDialogDescription() }</ui.CardDescription>
		</ui.CardHeader>
		<ui.CardContent>
			<ui.Dialog>
				<ui.Button
					variant="outline"
					data-gsxui-slot-dialog-trigger
					aria-haspopup="dialog"
					aria-expanded="false"
					hx-get={URL(ctx, "/fragments/server-info")}
					hx-target="#server-info"
				>
					{ M(ctx).HomeDialogOpen() }
				</ui.Button>
				<ui.DialogContent hideCloseButton={true}>
					<ui.DialogHeader>
						<ui.DialogTitle>{ M(ctx).HomeDialogHeading() }</ui.DialogTitle>
						<ui.DialogDescription>{ M(ctx).HomeDialogSubtitle() }</ui.DialogDescription>
					</ui.DialogHeader>
					<div id="server-info" class="text-sm text-muted-foreground">{ M(ctx).HomeDialogLoading() }</div>
					<ui.DialogFooter showCloseButton={false}>
						<ui.Button variant="outline" data-gsxui-dialog-close data-gsxui-slot-dialog-footer-close>
							{ M(ctx).HomeDialogClose() }
						</ui.Button>
					</ui.DialogFooter>
				</ui.DialogContent>
			</ui.Dialog>
		</ui.CardContent>
	</ui.Card>
}

component TabsCard() {
	<ui.Card>
		<ui.CardHeader>
			<ui.CardTitle>{ M(ctx).HomeTabsTitle() }</ui.CardTitle>
			<ui.CardDescription>{ M(ctx).HomeTabsDescription() }</ui.CardDescription>
		</ui.CardHeader>
		<ui.CardContent>
			<ui.Tabs value="overview">
				<ui.TabsList>
					<ui.TabsTrigger value="overview" selected>{ M(ctx).HomeTabsOverview() }</ui.TabsTrigger>
					<ui.TabsTrigger
						value="stats"
						hx-get={URL(ctx, "/fragments/stats")}
						hx-target="#tab-stats"
						hx-trigger="click once"
					>
						{ M(ctx).HomeTabsStats() }
					</ui.TabsTrigger>
				</ui.TabsList>
				<ui.TabsContent value="overview" selected class="pt-3 text-sm">
					{ M(ctx).HomeTabsOverviewBody() }
				</ui.TabsContent>
				<ui.TabsContent value="stats" class="pt-3 text-sm">
					<div id="tab-stats" class="text-muted-foreground">{ M(ctx).HomeTabsLoading() }</div>
				</ui.TabsContent>
			</ui.Tabs>
		</ui.CardContent>
	</ui.Card>
}

// LiveCard uses the htmx 4 hx-live extension: state lives in the DOM (data-count), :text re-renders after
// mutations, and q() queries relative to the element (patterns from jackielii/hx-live-vs-alpine).
component LiveCard(components []string) {
	<ui.Card>
		<ui.CardHeader>
			<ui.CardTitle>{ M(ctx).HomeLiveTitle() }</ui.CardTitle>
			<ui.CardDescription>{ M(ctx).HomeLiveDescription() }</ui.CardDescription>
		</ui.CardHeader>
		<ui.CardContent class="flex flex-col gap-4">
			<div data-count="0" class="flex items-center gap-3">
				<ui.Button variant="outline" hx-on:click=js`data.count++`>{ M(ctx).HomeLiveIncrement() }</ui.Button>
				<span class="text-sm">
					{ M(ctx).HomeLiveCount() }
					<ui.Badge variant="secondary" :text=js`data.count`>0</ui.Badge>
				</span>
			</div>
			<div class="flex flex-wrap items-start gap-3">
				<div data-open="false" class="relative" hx-on="click from:outside -> data.open = false">
					<ui.Button variant="outline" hx-on:click=js`data.open = !data.open`>{ M(ctx).HomeLiveMenu() }</ui.Button>
					<div
						hidden
						:hidden=js`!data.open`
						class="absolute z-10 mt-2 flex w-40 flex-col gap-1 rounded-md border bg-popover p-2 text-sm shadow-md"
					>
						<span>{ M(ctx).HomeLiveStateBag() }</span>
						<code>data.open</code>
					</div>
				</div>
				<ui.Button
					variant="outline"
					aria-pressed="false"
					hx-on:click=js`aria.pressed = !aria.pressed`
					:class=js`{ 'font-bold underline': aria.pressed }`
				>
					{ M(ctx).HomeLiveBold() }
				</ui.Button>
			</div>
			<div class="flex flex-col gap-2">
				<ui.Input placeholder={M(ctx).HomeLiveFilterPlaceholder()} aria-label={M(ctx).HomeLiveFilterLabel()}/>
				<ul
					class="flex flex-wrap gap-2"
					translate="no"
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

component BoardCard() {
	<ui.Card>
		<ui.CardHeader>
			<ui.CardTitle>{ M(ctx).HomeBoardTitle() }</ui.CardTitle>
			<ui.CardDescription>{ M(ctx).HomeBoardDescription() }</ui.CardDescription>
		</ui.CardHeader>
		<ui.CardContent>
			<ui.Button href={URL(ctx, "/board")}>
				<icon.Radio/>
				{ M(ctx).HomeBoardOpen() }
			</ui.Button>
		</ui.CardContent>
	</ui.Card>
}
