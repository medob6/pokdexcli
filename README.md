# Pokedex (Go + React)


This project contains a Pokedex CLI (Go) with optional TUI and a small React frontend showing Pokémon cards.


## Features
- CLI REPL with commands: `map`, `mapb`, `explore`, `catch`, `inspect`, `pokedex`, `help`, `exit`
- Disk cache for API responses
- Persistent store saved to `.pokedex/store.json`
- Optional Bubble Tea TUI for visual terminal interface
- Optional Gin proxy for the Pokémon TCG API
- React frontend to display card images (uses proxy if configured)


## Run the CLI (Go)


Requirements: Go 1.20+ (or your installed version)


```bash
# from project root
cd cmd/pokedex-cli
go run .