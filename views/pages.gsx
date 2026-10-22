package views

// Each page's Page method (pages.go's tree): structpages renders it with what the page's Props returned.

component (h Home) Page(p HomeProps) {
	<HomePage target={p.Target} env={p.Env} flavours={p.Flavours} components={p.Components}/>
}

component (b Board) Page(p BoardProps) {
	<BoardPage topic={p.Topic} b={p.State} live={p.Live}/>
}

component (f Formats) Page(conn Connection) {
	<FormatsPage conn={conn}/>
}

component (c Components) Page() {
	<ComponentsPage/>
}

component (a About) Page(stack []StackItem) {
	<AboutPage stack={stack}/>
}
