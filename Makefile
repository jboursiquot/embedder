DATA_DIR ?= data
URLS = \
	https://gutenberg.org/cache/epub/2680/pg2680.txt

.PHONY: download_books
download_books:
	@mkdir -p $(DATA_DIR)
	@for url in $(URLS); do \
		curl -o $(DATA_DIR)/$$(basename $$url) -L $$url; \
	done

tidy:
	go mod tidy

pull-ollama:
	docker pull ollama/ollama:latest

run-ollama:
	docker run --rm --name ollama \
		-v ./data/ollama:/root/.ollama \
		-p 11434:11434 \
		ollama/ollama:latest

pull-ollama-model:
	docker exec -it ollama ollama pull all-minilm

pull-pgvector:
	docker pull pgvector/pgvector:pg16

run-pgvector:
	docker run --rm --name pgvector \
		-e POSTGRES_USER=postgres \
		-e POSTGRES_PASSWORD=password \
		-e POSTGRES_DB=test 
		-v ./data/pgvector:/var/lib/postgresql/data \
		-p 5432:5432 \
		pgvector/pgvector:pg16

run:
	go run *.go