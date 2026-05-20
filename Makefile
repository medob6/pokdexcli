.PHONY: cli proxy frontend dev stop

cli:
	cd cmd/pokedex-cli && go run .

proxy:
	cd web/proxy && go run .

frontend:
	cd frontend && npm install && npm run dev

# Starts proxy and frontend together (proxy in background)
dev:
	cd web/proxy && go run . & \
	cd frontend && npm install && npm run dev

# Stops proxy and frontend (best effort)
stop:
	-@pkill -f "web/proxy" || true
	-@pkill -f "vite" || true
