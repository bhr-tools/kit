# Kit

Shared pieces of the bhr-tools web apps (Go + SQLite + vanilla JS). It shares
*how* things work, never *what* an app shows: no screens, no business
components.

- **Layer 1, invisible base**: server start-up, HTTP helpers, auth, numbered
  migrations, attachments, offline sync, DOM helpers.
- **Layer 2, interaction primitives**: dialogs and sheets, keyboard tables with
  `?` help, undo, drag gestures, selection, swipes, phone navigation, lock.
  Same behaviour everywhere, no imposed look.
- **Layer 3, tokens**: `web/css/base.css` — Rosé Pine (main / Dawn), roles
  (`--bg`, `--text`, `--accent`…) and scales (`--r-*`, `--fs-*`…). Kit
  components read only the roles; each app overrides them to keep its own
  identity.

## Use in an app

    go get github.com/bhr-tools/kit && go mod vendor

The browser files are embedded in the module (`web.Files`) and served under
`/kit/`: `@import url("/kit/css/kit.css");` at the top of the app's CSS, then its overrides.

## Tests

    go test ./...
    node --test "tests/js/*.test.mjs"

An app's Node tests reach the kit by URL like the browser:

    node --import ./vendor/github.com/bhr-tools/kit/web/node/register.mjs --test "tests/js/*.test.mjs"

## Demo and browser tests

`cmd/demo` serves a page using every primitive (keys and ? help, undo,
dialogs, menus, drag to reorder or onto a folder, phone pages and sheets, tab
bar, lock screen, offline service worker):

    go run ./cmd/demo -no-auth        # http://127.0.0.1:8099
    tests/e2e/run.sh                  # Playwright from nixpkgs, no Docker
