package views

// NowFragment answers the home page's hx-get with the server time and the compiler that served it.
component NowFragment(now string, target string) {
	<time datetime={now}>{ now }</time>{ " from " }<code>{ target }</code>
}

// GreetFragment answers the home page's form post.
component GreetFragment(name string) {
	{ if name == "" {
		<p class="text-destructive">Please enter a name.</p>
	} else {
		<p>Hello, { name }.</p>
	} }
}
