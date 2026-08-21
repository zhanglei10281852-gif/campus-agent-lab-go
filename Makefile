.PHONY: test race vet build measure frontend-install frontend-test frontend-typecheck frontend-build verify

test:
	cd backend && go test ./... -count=1

race:
	cd backend && go test -race ./... -count=1

vet:
	cd backend && go vet ./...

build:
	cd backend && go build ./...

measure:
	go run ./scripts/measure_project.go -root . -frontend-roots frontend-admin -enforce

frontend-install:
	cd frontend-admin && npm ci

frontend-test:
	cd frontend-admin && npm test -- --run

frontend-typecheck:
	cd frontend-admin && npm run typecheck

frontend-build:
	cd frontend-admin && npm run build

verify: test race vet build frontend-test frontend-typecheck frontend-build
