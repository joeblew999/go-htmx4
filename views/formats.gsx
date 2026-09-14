package views

import "github.com/joeblew999/go-htmx4/ui"

// Formats shows kit/i18n formatting in the request's locale: the same options JavaScript's Intl takes, with
// output computed on the server (byte-identical to Intl, see kit/i18n's conformance test).
component Formats() {
	<Layout title="Formats" path="/formats">
		<div class="flex flex-col gap-2">
			<h1 class="text-3xl font-semibold tracking-tight">Formats</h1>
			<p class="text-muted-foreground">
				Locale-aware formatting rendered by Go from Unicode CLDR, with the options and output of JavaScript's Intl.
				Switch language at the bottom of the page.
			</p>
		</div>
		<LocaleCard facts={Facts(ctx)}/>
		{ for _, sec := range NumberSections(ctx) {
			<ui.Card>
				<ui.CardHeader>
					<ui.CardTitle>{ sec.Title }</ui.CardTitle>
					<ui.CardDescription>{ sec.Description }</ui.CardDescription>
				</ui.CardHeader>
				<ui.CardContent>
					<ui.Table>
						<ui.TableHeader>
							<ui.TableRow>
								<ui.TableHead>Input</ui.TableHead>
								<ui.TableHead>Options</ui.TableHead>
								<ui.TableHead class="text-end">Result</ui.TableHead>
							</ui.TableRow>
						</ui.TableHeader>
						<ui.TableBody>
							{ for _, r := range sec.Rows {
								<ui.TableRow>
									<ui.TableCell class="font-medium">
										<bdi>{ r.Label }</bdi>
									</ui.TableCell>
									<ui.TableCell>
										<code dir="ltr" class="text-xs text-muted-foreground">{ r.Options }</code>
									</ui.TableCell>
									<ui.TableCell class="text-end tabular-nums">
										<bdi>{ r.Value }</bdi>
									</ui.TableCell>
								</ui.TableRow>
							} }
						</ui.TableBody>
					</ui.Table>
				</ui.CardContent>
			</ui.Card>
		} }
	</Layout>
}

component LocaleCard(facts LocaleFacts) {
	<ui.Card>
		<ui.CardHeader>
			<ui.CardTitle>{ facts.NativeName }</ui.CardTitle>
			<ui.CardDescription>{ facts.CLDR }</ui.CardDescription>
		</ui.CardHeader>
		<ui.CardContent class="flex flex-wrap gap-2">
			<ui.Badge variant="secondary">{ facts.ID }</ui.Badge>
			<ui.Badge variant="outline">{ facts.Maximal }</ui.Badge>
			<ui.Badge variant="outline">dir { facts.Dir }</ui.Badge>
			<ui.Badge variant="outline">numbers { facts.NumberingSystem }</ui.Badge>
			{ if facts.NativeSystem != "" {
				<ui.Badge variant="outline">native { facts.NativeSystem }</ui.Badge>
			} }
			<ui.Badge variant="outline">{ facts.Currency }</ui.Badge>
		</ui.CardContent>
	</ui.Card>
}
