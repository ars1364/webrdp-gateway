# Local CI gate: the same checks GitHub Actions runs (minus image scans).
.PHONY: ci guards backend frontend integration

ci: guards backend frontend

guards:
	scripts/ci/file-size.sh
	scripts/ci/handler-tests.sh
	scripts/ci/migrations.sh
	scripts/ci/docs-drift.sh
	scripts/ci/airgap.sh
	scripts/ci/backend-rules.sh
	scripts/ci/nginx-headers.sh

backend:
	cd backend && test -z "$$(gofmt -l .)" && go vet ./... && go test -race -count=1 ./...

frontend:
	cd frontend && npx tsc --noEmit && npm run lint && npm test && npm run build

# needs a throwaway Postgres: TEST_DATABASE_URL=postgres://... make integration
integration:
	cd backend && go test -tags=integration -count=1 -p 1 ./internal/store/
