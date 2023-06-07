PG_ADDR ?= 'postgres://ps_user:ps_password@localhost:5432/backend?sslmode=disable'
COMMIT_SHA := $(shell git rev-parse --short HEAD)
export DB_CONN="user=ps_user password=ps_password dbname=backend sslmode=disable host=localhost"

.PHONY: build
build:
	rm -rf dist
	mkdir dist
	go build -o dist ./...

.PHONY: deps
deps:
	docker-compose up -d

.PHONY: migrate
migrate:
	./run-migrate.sh

.PHONY: run
run: build
	./dist/api

.PHONY: deps-down
deps-down:
	docker-compose down

.PHONY: test
test:
	go test ./...

.PHONY: integration
integration: build
	docker-compose up -d
	./wait-for-postgres.sh 127.0.0.1 5432
	./run-migrate.sh
	go test -tags=integration ./...

.PHONY: docker-image
docker-image:
	docker rmi -f game-student-go:$(COMMIT_SHA)
	docker rmi -f 676187242411.dkr.ecr.us-east-1.amazonaws.com/game-student-go:$(COMMIT_SHA)
	docker container prune -f
	docker build -t game-student-go:$(COMMIT_SHA) .
	docker tag game-student-go:$(COMMIT_SHA) 676187242411.dkr.ecr.us-east-1.amazonaws.com/game-student-go:$(COMMIT_SHA)
	docker tag game-student-go:$(COMMIT_SHA) 676187242411.dkr.ecr.us-east-1.amazonaws.com/game-student-go:latest
	docker image prune -f

.PHONY: docker-run
docker-run:
	docker run --network=game_student -d -p 8080:8080 --platform linux/amd64 -ti game-student-go:$(COMMIT_SHA) /api

.PHONY: docker-stop
docker-stop:
	docker stop $$(docker ps -q)
	docker container prune -f

.PHONY: setup
setup:
	go install github.com/matryer/moq@latest

.PHONY: mocks
mocks:
	moq -out internal/database/mock_database.go internal/database Database
