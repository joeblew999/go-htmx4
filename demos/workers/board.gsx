package main

// BoardPage is the shared board page. It connects to /live/{topic} with hx-ws; the Room Durable
// Object pushes BoardFragment on every change.
//
// A deploy drops every socket at once (measured). hx-ws's defaults (500 ms ±30%) would bring 1,000
// tabs back within ~0.3 s, over a Room's ~1,000 requests/s soft limit, so the htmx-config meta
// spreads reconnects over 1–3 s.
component BoardPage(topic string, b Board) {
	<!DOCTYPE html>
	<html lang="en">
		<head>
			<meta charset="utf-8"/>
			<meta name="viewport" content="width=device-width, initial-scale=1"/>
			<title>Shared board · htmx 4 on Workers</title>
			<meta name="htmx-config" content="ws.reconnectDelay:2s ws.reconnectJitter:0.5"/>
			<link rel="stylesheet" href="/demo.css"/>
			<script src="/htmx.min.js"></script>
			<script src="/hx-ws.js"></script>
			<script>
				// HTTP responses and hx-ws pushes both go through htmx.swap and can arrive out of order.
				// Drop any swap whose board version is older than the one on screen.
				document.addEventListener("htmx:before:swap", (event) => {
					const incoming = /data-version="(\d+)"/.exec(event.detail.ctx.text ?? "");
					const current = document.getElementById("board")?.dataset.version;
					if (incoming && current && Number(incoming[1]) < Number(current)) event.preventDefault();
				});
			</script>
		</head>
		<body>
			<main>
				<h1>Shared board</h1>
				<p class="meta"><span id="presence">connecting…</span></p>
				<p class="meta">
					Every tab on topic <code>{ topic }</code> sees changes live: Go writes to D1, then the topic's Durable Object
					pushes the fragment to all tabs over hx-ws. <a href="/">← demo</a>
				</p>
				<div hx-ws:connect={"/live/" + topic} hx-swap="none">
					<BoardFragment b={b}/>
				</div>
				<section>
					<h2>Change it</h2>
					<form hx-post={"/board/add?topic=" + topic} hx-swap="none" class="inline">
						<input type="hidden" name="delta" value="-1"/>
						<button>−1</button>
					</form>
					<form hx-post={"/board/add?topic=" + topic} hx-swap="none" class="inline">
						<input type="hidden" name="delta" value="1"/>
						<button>+1</button>
					</form>
					<form hx-post={"/board/note?topic=" + topic} hx-swap="none" hx-on:htmx:after:request=js`this.reset()`>
						<input name="body" maxlength="280" placeholder="Leave a note" autocomplete="off" required/>
						<button>Post</button>
					</form>
				</section>
			</main>
		</body>
	</html>
}

// BoardFragment is the #board element as an out-of-band swap: the initial page, the poster's
// response and the hx-ws push all use it. Its data-version lets the page drop stale swaps, and the
// Room drops stale publishes, so its shape is the wire format (see TestBoardFragmentWireFormat).
component BoardFragment(b Board) {
	<section id="board" hx-swap-oob="true" data-version={b.Version}>
		<p class="value"><output>{ b.Value }</output></p>
		<ol class="notes">
			{ for _, n := range b.Notes {
				<li><time>{ n.CreatedAt }</time> { n.Body }</li>
			} }
		</ol>
		<p class="meta">topic <code>{ b.Topic }</code> · version { b.Version }</p>
	</section>
}
