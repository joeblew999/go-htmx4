package views

import (
	"github.com/joeblew999/go-htmx4/locales"
	"github.com/joeblew999/go-htmx4/ui"
)

// About lists the stack, and gives the boosted nav a third page to morph to. Names are brands (translate="no");
// roles are translated (StackItem.Role is a message).
component AboutPage(stack []StackItem) {
	<Layout title={locales.AboutTitle} description={locales.AboutDescription}>
		<h1 class="text-3xl font-semibold tracking-tight">{ locales.AboutHeading }</h1>
		<ui.Card>
			<ui.CardHeader>
				<ui.CardTitle>{ locales.AboutStack }</ui.CardTitle>
				<ui.CardDescription>{ locales.AboutStackDescription }</ui.CardDescription>
			</ui.CardHeader>
			<ui.CardContent class="flex flex-col gap-3">
				{ for i, s := range stack {
					{ if i > 0 {
						<ui.Separator/>
					} }
					<div class="flex items-center justify-between gap-4 text-sm">
						<a href={s.URL} class="font-medium underline-offset-4 hover:underline" translate="no">{ s.Name }</a>
						<span class="text-muted-foreground">{ s.Role }</span>
					</div>
				} }
			</ui.CardContent>
		</ui.Card>
	</Layout>
}
