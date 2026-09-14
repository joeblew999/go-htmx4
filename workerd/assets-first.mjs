// Local stand-in for Workers Static Assets under plain workerd (no wrangler, no Node).
//
// Serve a file from dist/site/ when one exists, otherwise hand the request to the Go Worker:
// what Cloudflare does in production with the default run_worker_first = false.
// Pattern from workerd's samples/static-files-from-disk (a worker in front of a disk service).
// Not deployed: on Cloudflare, dist/site/ is uploaded as Static Assets instead.

const TYPES = {
  css: "text/css; charset=utf-8",
  html: "text/html; charset=utf-8",
  ico: "image/x-icon",
  js: "text/javascript; charset=utf-8",
  png: "image/png",
  svg: "image/svg+xml",
  txt: "text/plain; charset=utf-8",
  woff2: "font/woff2",
};

export default {
  async fetch(req, env) {
    const { pathname } = new URL(req.url);
    const name = pathname.split("/").pop();
    // Only paths that look like files; "/" and routes like /fragments/now go to Go.
    if ((req.method === "GET" || req.method === "HEAD") && name.includes(".")) {
      const file = await env.ASSETS.fetch("http://assets" + pathname, { method: req.method });
      if (file.ok) {
        const type = TYPES[name.split(".").pop().toLowerCase()] ?? "application/octet-stream";
        return new Response(file.body, { headers: { "Content-Type": type } });
      }
    }
    return env.APP.fetch(req);
  },
};
