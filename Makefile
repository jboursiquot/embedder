.PHONY: default help download_books tidy pull-ollama run-ollama pull-ollama-model pull-pgvector run-pgvector run

default: help

help: ## show help message
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m\033[0m\n"} /^[$$()% a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

DATA_DIR ?= data
URLS = \
	https://gutenberg.org/cache/epub/2680/pg2680.txt

.PHONY: download_books
download_books: ## download books
	@mkdir -p $(DATA_DIR)
	@for url in $(URLS); do \
		curl -o $(DATA_DIR)/$$(basename $$url) -L $$url; \
	done

tidy: ## go mod tidy
	go mod tidy

pull-ollama: ## pull ollama docker image
	docker pull ollama/ollama:latest

run-ollama: ## run ollama docker container
	docker run --rm --name ollama \
		-v ./data/ollama:/root/.ollama \
		-p 11434:11434 \
		ollama/ollama:latest

try-ollama-embed:
	curl http://localhost:11434/api/embed -d '{"model": "all-minilm","input": "Why is the sky blue?"}'

pull-ollama-model: ## pull ollama model
	docker exec -it ollama ollama pull all-minilm

pull-pgvector: ## pull pgvector docker image
	docker pull pgvector/pgvector:pg17

run-pgvector: ## run pgvector docker container
	docker run --rm --name pgvector \
		-e POSTGRES_USER=postgres \
		-e POSTGRES_PASSWORD=password \
		-e POSTGRES_DB=test \
		-v ./data/pgvector:/var/lib/postgresql/data \
		-p 5432:5432 \
		pgvector/pgvector:pg17

run: ## run
	go run *.go -book=./data/pg2680.txt