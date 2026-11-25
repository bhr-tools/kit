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
`/kit/`: `@import url("/kit/css/base.css");` at the top of the app's CSS.

## Tests

    go test ./...
    node --test "tests/js/*.test.mjs"
