.PHONY: all build test sqlc dev-backend dev-frontend clean

all: build test

# Generate sqlc Go models from SQL schemas
sqlc:
	cd backend && sqlc generate

# Build Go Backend Server and Local Agent CLI
build: sqlc
	cd backend && go build -o bin/server.exe ./cmd/server
	cd agent && go build -o bin/apisentinel.exe ./cmd/apisentinel
	npm run build --workspace=apisentinel-frontend

# Run full test suite
test:
	cd backend && go test -v ./...
	npm run test --workspace=apisentinel-frontend

# Run Backend in Development Mode
dev-backend:
	cd backend && go run ./cmd/server/main.go

# Run Next.js Frontend
dev-frontend:
	npm run dev --workspace=apisentinel-frontend

clean:
	rm -rf backend/bin agent/bin
