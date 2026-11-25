# How it ships

wrangler deploys the Worker; `go tool charter deploy <environment>` applies the D1 migrations first, then runs it.
`wrangler.jsonc` has the environments.

| Environment | Command | Where |
|---|---|---|
| Local | `go tool charter preview` | http://localhost:8913, local D1 and Durable Objects |
| Staging | `go tool charter deploy staging` | https://go-htmx4-staging.gedw99.workers.dev, its own D1, not indexed |
| Production | `go tool charter deploy production` | https://go-htmx4.ubuntusoftware.net |

## Before a deploy

`go tool charter check`, and `go tool charter test e2e` after a change to pages, the board or browser code.

## Credentials

wrangler reads `CLOUDFLARE_API_TOKEN` and `CLOUDFLARE_ACCOUNT_ID` from the environment. They come from fnox, never
from a file in the repo:

```sh
fnox exec -- go tool charter deploy staging
```

## After a deploy

```sh
BASE=https://go-htmx4-staging.gedw99.workers.dev go test -C e2e -count=1 ./smoke
go tool charter test e2e -base https://go-htmx4-staging.gedw99.workers.dev
go tool charter seo audit      # production, as Googlebot reads it
```

## The database

A schema change is a new idempotent file in `migrations/`. wrangler records which have run in each database.
`board.version` never goes backwards: never delete `board` rows or reset versions.
