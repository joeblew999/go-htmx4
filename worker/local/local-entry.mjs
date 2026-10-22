// Local-only entry for workerd (never deployed). workerd has no D1 without miniflare, so this wraps
// index.mjs and hands the Go Worker a D1-shaped `DB` backed by the LocalD1 Durable Object (SQLite).
// The Go code path, database/sql + workers-go's d1 driver, is the same as on Cloudflare. workerd has no
// rate limiting binding either, so WRITES is an in-memory fixed window with the deployed settings.

import entry from "../index.mjs";
import { d1Shim } from "./local-d1.mjs";

export { Room } from "../room.mjs";
export { LocalD1 } from "./local-d1.mjs";

// Same settings as the deploy's -ratelimit WRITES=…:60/10 (ci/deploy.sh). Same API: limit({key}) → {success}.
function localRateLimit({ limit, period }) {
  const windows = new Map();
  return {
    async limit({ key }) {
      const now = Date.now();
      const w = windows.get(key);
      if (!w || now - w.start >= period * 1000) {
        windows.set(key, { start: now, count: 1 });
        return { success: true };
      }
      w.count++;
      return { success: w.count <= limit };
    },
  };
}
const writes = localRateLimit({ limit: 60, period: 10 });

export default {
  fetch(request, env, ctx) {
    // Cloudflare gives the Worker each visitor's zone from their IP (request.cf.timezone). Locally there is no
    // Cloudflare: mise run serve and dev pass the developer's own zone, so the first page is in it, as in production.
    if (env.LOCAL_TIME_ZONE && !request.cf?.timezone) {
      request = new Request(request, { cf: { ...request.cf, timezone: env.LOCAL_TIME_ZONE } });
    }
    return entry.fetch(request, { ...env, DB: env.DB ?? d1Shim(env.LOCAL_D1), WRITES: env.WRITES ?? writes }, ctx);
  },
};
