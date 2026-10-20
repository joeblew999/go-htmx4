# 2026-10-20_1746_previews-done-properly.md

Finish the preview pipeline as one change, decided up front: correct, reproducible from the repo, fast, and
cleaned up. Today's pipeline (PR #5) works but was built step by step; this plan replaces that with a design.

## Where it stands (main, afca3bf)

- Push → GitHub CI (`check.yml` → `ci/check.sh`): build + test, then publish `<sha>.tar.gz` (wasm, `dist/site`,
  static `bin/deploy`) to the `preview-builds` pre-release.
- Workers Builds → `ci/workers-builds.sh`: wait for that file, download, deploy the preview with the prebuilt tool
  (~6 s). Fallbacks: compile the deploy tool with Go; no GitHub build in 8 min (or `PREVIEW_BUILD=compile`) →
  full TinyGo build on Cloudflare.
- Measured (1a75487): GitHub publishes ~3.5 min after the push; preview live 6 s later.

## Problems, all of them

1. **The Cloudflare side isn't in the repo.** Repo connection, the branch trigger, its commands, the
   `CLOUDFLARE_ACCOUNT_ID` variable, the do-nothing `main` trigger: set by hand through the API. A rename, a
   second machine or a wiped setting can't reproduce it.
2. **Borrowed build token** (mailflare's). The `worker-go` one I tried first was dead.
3. **Logic in untested bash:** waiting, downloading, verifying, falling back, pruning assets, picking the tool.
4. **The repo name is hard-coded** (`joeblew999/go-htmx4`) in `ci/workers-builds.sh`: breaks `mise run rename`.
5. **Previews are never deleted.** Workers Builds doesn't run when a branch is deleted, and GitHub has no
   Cloudflare credentials, so previews for merged branches (3 already) live forever, each with its own Durable
   Object storage.
6. **The preview URL isn't on the PR.** Only in Cloudflare's build log.
7. **Docs are wrong already:** README says Cloudflare builds with `mise install` / `mise run build` (that was PR #4).
8. **The live site still runs the pre-update build.** Production deploys need fnox (Gerard's Mac).
9. **The wait is GitHub CI**: 41 s tools + ~110 s TinyGo + smoke/tests before it publishes.

## Decisions and their ramifications

1. **One Go tool owns the whole preview flow** — `cmd/deploy` gains `-preview-from-release <sha>` (logic in a new
   `kit/cfdeploy` file; cmd stays flags only): wait for the asset, download, verify (the wasm's size and the asset
   count match a manifest CI writes into the tarball), deploy. Tested against fake GitHub and Cloudflare APIs like
   the rest of `kit/`.
   - Ramification: the *first* download can't be done by that tool (it's inside the download). So bash keeps
     exactly one job: fetch the tool. CI publishes `bin/deploy` as its own small per-commit asset next to the
     tarball; `ci/workers-builds.sh` becomes ~10 lines: install nothing, curl the tool, exec it. Fallback (no tool
     after the wait): today's mise + Go path, unchanged.
   - Publishing from CI moves into Go too (`-publish-preview-build`), using the GitHub REST API with the built-in
     token (no `gh` dependency); pruning to 40 assets goes with it.
2. **The Cloudflare side becomes code:** `mise run cf:builds` (Go, `kit/cfdeploy`) creates or updates the repo
   connection, the branch trigger, its commands and build variables, idempotently, from `mise.toml`
   (`APP_NAME`, a new `APP_REPO`). Run it once from anywhere with Cloudflare credentials; `mise run rename` sets
   `APP_REPO`.
   - Ramification: the API can't create a build token (it needs an API token secret). The task takes the build
     token by *name* and fails with the one manual step if it's missing: create "go-htmx4 build token" in the
     dashboard once. Until then it keeps using mailflare's, said so in the task output.
   - It also removes the do-nothing `main` trigger (decision 6) and the leftover Previews-mode setting.
3. **Repo name from config, not code:** `APP_REPO` in `mise.toml` `[env]`, passed to Workers Builds as a build
   variable by `cf:builds`. Fixes problem 4 and keeps rename working.
4. **Preview cleanup without secrets:** every preview deploy also prunes. The deploy tool lists the Worker's
   previews and deletes those whose branch no longer exists (`git ls-remote` on the public repo; Cloudflare's
   builders reach github.com, proven by the downloads). Merged branches lose their preview at the next push to any
   branch.
   - Ramification: branches still on GitHub keep their previews (4 stale merged branches exist now; this sandbox
     can't delete branches, the proxy blocks it). Turning on "automatically delete head branches" fixes that going
     forward: in the GitHub settings command already given (`delete_branch_on_merge`).
   - `main` never has a preview, so it's never pruned.
5. **The preview URL is known before the deploy, so CI posts it.** We choose the preview name, so we make it a
   valid slug ourselves (lowercase, `[a-z0-9-]`, from the branch), and the URL is
   `https://<slug>-<APP_NAME>.<subdomain>.workers.dev`. CI comments it on the PR with the built-in token (one
   comment, edited per push). The workers.dev subdomain goes in `mise.toml` (`CF_SUBDOMAIN=gedw99`).
   - Ramification: the comment appears before the preview is live (~seconds). It says "deploying" and links the
     Workers Builds check, which goes green when live.
6. **Production stays manual for now** (`mise run deploy` on the Mac): auto-deploying `main` from Cloudflare would
   need its `search:audit`/`token-check` steps reworked (they use fnox) and is a separate decision. The live site
   gets deployed the next time Gerard runs it; until then it's behind `main`.
7. **Faster GitHub CI — only safe wins:**
   - Publish right after `build`, before smoke and the Go tests, in parallel with them. The preview appears
     ~30 s earlier. Ramification: a commit that fails its tests still gets a preview; CI is red on the PR. Previews
     are for looking, CI is the gate, so that's acceptable.
   - Not doing: caching between CI runs (needs `actions/cache`, a JavaScript action; no-Node rule), lowering
     TinyGo optimisation (the preview would differ from what production deploys). TinyGo (~110 s) stays the floor.
8. **Docs:** README and AGENTS.md describe the real flow; the previews plan is closed and moved to `done/`.

## Out of scope (separate, already planned)

gsxui refresh + i18n of its strings, chromedp migration, auto-merge/branch protection (GitHub settings), production
auto-deploy.

## Steps (one branch, one PR)

- [ ] `kit/cfdeploy`: preview-from-release (wait, download, verify, deploy), publish + prune assets (GitHub REST),
      prune previews of deleted branches, preview slug, Workers Builds config (connection, trigger, variables);
      all tested with fakes
- [ ] `cmd/deploy` flags; `mise run cf:builds`; `APP_REPO`, `CF_SUBDOMAIN` in `mise.toml`; rename updates them
- [ ] `ci/check.sh`: publish after build (Go), PR comment; `ci/workers-builds.sh` down to fetch-tool-and-exec
- [ ] `mise run check` + `e2e` locally, as CI runs them
- [ ] Push; verify on a real PR: comment, preview live, prune removes the merged branches' previews
- [ ] Run `cf:builds` against the account (replaces the hand-made config); verify the next push still deploys
- [ ] README, AGENTS.md; close the previews plan
