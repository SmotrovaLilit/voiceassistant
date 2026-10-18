.DEFAULT_GOAL := run

.PHONY: up run down

up:
	docker compose -f deploy/docker-compose.yml up -d --wait --wait-timeout 1200

run: up
	$(MAKE) -C applications/voiceassistant-api run

down:
	docker compose -f deploy/docker-compose.yml down
