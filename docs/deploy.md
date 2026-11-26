# How it ships

wrangler deploys the Worker; `go tool charter deploy <environment>` applies the D1 migrations first, then runs it.
`wrangler.jsonc` has the environments.

| Environment | Command | Where |
|---|---|---|
| Local | `go tool charter preview` | http://localhost:8913, local D1 and Durable Objects |
| Staging | `go tool charter deploy staging` | https://go-htmx4-staging.gedw99.workers.dev, its own D1, not indexed |
| Production | `go tool charter deploy production` | https://go-htmx4.ubuntusoftware.net |

## Before a deploy

`go tool charter check && go tool charter test`, and `go tool charter test e2e` after a change to pages, the board or browser code.

## Credentials

wrangler reads `CLOUDFLARE_API_TOKEN` and `CLOUDFLARE_ACCOUNT_ID` from the environment. They come from fnox, never
from a file in the repo:

```sh
fnox exec -- go tool charter deploy staging
```

## After a deploy

```sh
go tool charter test --env staging        # the smoke tests against the deployed Worker
go tool charter test e2e --env staging    # the browser tests
go tool charter run audit      # production, as Googlebot reads it
```

## The database

A schema change is a new idempotent file in `migrations/`. wrangler records which have run in each database.
`board.version` never goes backwards: never delete `board` rows or reset versions.
