package views

import "github.com/joeblew999/go-htmx4/ui"

// Formats shows Intl formatting in the request's locale: each row is an Intl call with its options, formatted by the
// viewer's own browser (static/intl.js) in the page's language and the viewer's time zone. The plural rules are the
// exception, computed on the server, since they choose the wording of translated messages.
component Formats() {
	<Layout title={M(ctx).FormatsTitle()} description={M(ctx).FormatsDescription()} path="/formats">
		<div class="flex flex-col gap-2">
			<h1 class="text-3xl font-semibold tracking-tight">{ M(ctx).FormatsTitle() }</h1>
			<p class="text-muted-foreground">{ M(ctx).FormatsIntro() }</p>
		</div>
		<LocaleCard facts={Facts(ctx)}/>
		{ for _, sec := range FormatSections(ctx) {
			<ui.Card>
				<ui.CardHeader>
					<ui.CardTitle>{ sec.Title }</ui.CardTitle>
					<ui.CardDescription>{ sec.Description }</ui.CardDescription>
				</ui.CardHeader>
				<ui.CardContent>
					<ui.Table>
						<ui.TableHeader>
							<ui.TableRow>
								<ui.TableHead>{ M(ctx).FormatsInput() }</ui.TableHead>
								<ui.TableHead>{ M(ctx).FormatsOptions() }</ui.TableHead>
								<ui.TableHead class="text-end">{ M(ctx).FormatsResult() }</ui.TableHead>
							</ui.TableRow>
						</ui.TableHeader>
						<ui.TableBody>
							{ for _, r := range sec.Rows {
								<ui.TableRow>
									<ui.TableCell class="font-medium">
										<code dir="ltr" translate="no">{ r.Label }</code>
									</ui.TableCell>
									<ui.TableCell>
										<code dir="ltr" class="text-xs text-muted-foreground">{ r.Options }</code>
									</ui.TableCell>
									<ui.TableCell class="text-end tabular-nums">
										<bdi
											{ if r.Kind != "" {
												data-intl={r.Kind}
												data-value={r.Value}
												{ if r.Opts != "" {
													data-opts={r.Opts}
												} }
												{ if r.End != "" {
													data-end={r.End}
												} }
												{ if r.Unit != "" {
													data-unit={r.Unit}
												} }
											} else {
												data-i18n="cldr"
											} }
										>
											{ r.Fallback }
										</bdi>
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
			<ui.CardTitle translate="no">{ facts.NativeName }</ui.CardTitle>
			<ui.CardDescription translate="no">{ facts.CLDR }</ui.CardDescription>
		</ui.CardHeader>
		<ui.CardContent class="flex flex-wrap gap-2">
			<ui.Badge variant="secondary" translate="no">{ facts.ID }</ui.Badge>
			<ui.Badge variant="outline" translate="no">{ facts.Maximal }</ui.Badge>
			<ui.Badge variant="outline">{ M(ctx).FormatsDirection(facts.Dir) }</ui.Badge>
			<ui.Badge variant="outline">{ M(ctx).FormatsNumbering(facts.NumberingSystem) }</ui.Badge>
			{ if facts.NativeSystem != "" {
				<ui.Badge variant="outline">{ M(ctx).FormatsNative(facts.NativeSystem) }</ui.Badge>
			} }
			<ui.Badge variant="outline" translate="no">{ facts.Currency }</ui.Badge>
			<ui.Badge variant="outline">{ M(ctx).FormatsTimeZone(TimeZone(ctx)) }</ui.Badge>
		</ui.CardContent>
	</ui.Card>
}
