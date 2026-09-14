# Custom domains for the two Workers

**Status:** **Parked** (2026-09-14: "forget custom domains for now"), not started · **Created:** 2026-09-14 09:15

## Goal

Serve `go-htmx4-workers-demo` and `go-htmx4-gsxui-demo` on hostnames in one of your Cloudflare zones instead of (or as well
as) `*.gedw99.workers.dev`, set up by our Go deploy tool: no dashboard clicks, no wrangler.

## Ground rules

- No Node, no wrangler; Cloudflare REST API via `demos/workers/cmd/deploy`; credentials via fnox.
- **Attaching a domain is outward-facing (DNS + certificate): needs your explicit OK and hostnames.**

## Upstream API

- Workers Custom Domains: `PUT /accounts/{account_id}/workers/domains` with `hostname`, `service` (script name),
  `zone_id` or `zone_name`, `environment`; `GET …/workers/domains` lists; `DELETE …/workers/domains/{domain_id}`
  removes (cloudflare-go `workers/domain.go`). Cloudflare creates the DNS record and certificate for the hostname.
- Zones this API token can see (read-only check, 2026-09-14): `amplify-cms.com`, `amplifycms.com`, `amplify-cms.net`,
  `amplifycms.net`, `amplify-cms.org`, `amplifycms.org`, `ubuntudesign.com`, `ubuntusoftware.net`.

## Plan

### Phase 1: Deploy tool

- [ ] `cmd/deploy -domain host.example.com` (repeatable): look up the zone by the hostname's registrable domain, `PUT`
      the custom domain for `-name`, idempotent on re-deploys (skip if already attached to this script, fail if attached
      to another). `-dry-run` prints the plan.
- [ ] Keep `workers.dev` enabled (or add `-workers-dev=false`), decided below.

### Phase 2: Attach (⚠ needs OK)

- [ ] Add the hostnames to `demo:workers:deploy` and `demo:gsxui:workers:deploy`; deploy; wait for the certificate.
- [ ] Remote smoke tests + wsload + browser checks against the custom hostnames (WebSockets over the custom domain too).

### Phase 3: Docs

- [ ] README demo URLs, AGENTS.md deploy notes.

## Open questions

- **Which zone and hostnames?** e.g. `htmx.<zone>` + `gsxui.<zone>`, or one host with two paths (not supported by custom
  domains; would need Routes instead).
- Keep the `*.workers.dev` URLs working as well?
