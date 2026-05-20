````markdown
# Pokedex (Go + React)


This project contains a Pokedex CLI (Go) with optional TUI and a small React frontend showing Pokemon cards.

## Quick Start (Recommended)

From the project root:

```bash
make dev
```

This starts the Go proxy and the React frontend together. When you are done:

```bash
make stop
```

Open the Vite URL printed in the terminal. The dev server proxies `/api` to the Go server.


## Features
- CLI REPL with commands: `map`, `mapb`, `explore`, `catch`, `inspect`, `pokedex`, `help`, `exit`
- Disk cache for API responses
- Persistent store saved to `.pokedex/store.json`
- Optional Bubble Tea TUI for visual terminal interface
- Optional Gin proxy for the Pokemon TCG API
- React frontend to display card images (uses proxy if configured)
- React frontend game mode: timed encounters, streak XP, and daily missions


## Run the CLI (Go)


Requirements: Go 1.20+ (or your installed version)


```bash
# from project root
cd cmd/pokedex-cli
go run .
```

Or use the one-liner from the project root:

```bash
make cli
```

## Run the Go API (for the frontend)

```bash
# from project root
cd web/proxy
go run .
```

Or use the one-liner from the project root:

```bash
make proxy
```

The API starts on http://localhost:8080.

## Run the React frontend

```bash
# from project root
cd frontend
npm install
npm run dev
```

Or use the one-liner from the project root:

```bash
make frontend
```

## Make Targets

From the project root:

```bash
make cli       # Run the Go CLI
make proxy     # Run the Go proxy API
make frontend  # Run the React app
make dev       # Run proxy + frontend together
make stop      # Stop proxy + frontend (best effort)
```

## Requirements

- Go 1.20+ (or your installed version)
- Node.js 18+ with npm

## Project Layout

- cmd/pokedex-cli: Go CLI entry point
- internal/: CLI logic, API client, persistence, and TUI
- web/proxy: Go proxy server for the Pokemon TCG API
- frontend: Vite + React UI

## Notes

- The proxy listens on http://localhost:8080
- The frontend dev server proxies `/api` to the Go server
- CLI progress is saved under `.pokedex/store.json`
````
