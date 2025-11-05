# pokdexcli

Lightweight Pokedex: CLI + a small web frontend.

This repository contains:
- cmd/pokedex-cli — CLI that talks to PokeAPI (https://pokeapi.co).
- web — a minimal Go web server + static frontend.

Quickstart

1) CLI
- Build and run the CLI:
  go run ./cmd/pokedex-cli

- Example usage in CLI:
  map
  explore 0
  catch pikachu
  pokedex
  inspect pikachu
  exit

The CLI persists caught pokemon to ./.pokedex/store.json.

2) Web (static frontend + simple proxy)
- Run the web server:
  go run ./web/main.go

- Open: http://localhost:8080

What the web server does:
- Serves files from web/static (index.html).
- Proxies requests from /api/* to https://pokeapi.co/api/v2/* to avoid CORS issues from the browser.

Notes
- The web frontend stores "caught" pokemon in the browser's localStorage (key: pokedex:caught). This keeps the server side minimal.
- If you want server-side persistence for the web UI, we can extend the server to save into the existing CLI store file (./.pokedex/store.json) or add a tiny embedded DB.

Branch and changes
- I prepared these files on branch: fix/web-server
