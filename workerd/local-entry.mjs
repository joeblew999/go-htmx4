// Local-only entry for workerd (never deployed). workerd has no D1 without miniflare, so this wraps
// index.mjs and hands the Go Worker a D1-shaped `DB` backed by the LocalD1 Durable Object (SQLite).
// The Go code path, database/sql + workers-go's d1 driver, is the same as on Cloudflare.

import entry from "../index.mjs";
import { d1Shim } from "./local-d1.mjs";

export { Room } from "../room.mjs";
export { LocalD1 } from "./local-d1.mjs";

export default {
  fetch(request, env, ctx) {
    return entry.fetch(request, { ...env, DB: env.DB ?? d1Shim(env.LOCAL_D1) }, ctx);
  },
};
