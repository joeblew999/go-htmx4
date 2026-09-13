package views

import "github.com/joeblew999/go-htmx4/demos/gsxui/ui"

type StackItem struct {
	Name, Role, URL string
}

// About is the second page, so the boosted nav has somewhere to morph to.
component About(stack []StackItem) {
	<Layout title="About" path="/about">
		<h1 class="text-3xl font-semibold tracking-tight">About this demo</h1>
		<ui.Card>
			<ui.CardHeader>
				<ui.CardTitle>Stack</ui.CardTitle>
				<ui.CardDescription>Everything is pinned in mise.toml. Nothing here needs Node.</ui.CardDescription>
			</ui.CardHeader>
			<ui.CardContent class="flex flex-col gap-3">
				{ for i, s := range stack {
					{ if i > 0 {
						<ui.Separator/>
					} }
					<div class="flex items-center justify-between gap-4 text-sm">
						<a href={s.URL} class="font-medium underline-offset-4 hover:underline">{ s.Name }</a>
						<span class="text-muted-foreground">{ s.Role }</span>
					</div>
				} }
			</ui.CardContent>
		</ui.Card>
	</Layout>
}
