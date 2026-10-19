.DEFAULT_GOAL := run

.PHONY: up run down request

AUDIO ?= audioexamples/anna-agile-robots.wav
API_URL ?= http://127.0.0.1:8081

up:
	docker compose -f deploy/docker-compose.yml up -d --wait --wait-timeout 1200

run: up
	$(MAKE) -C applications/voiceassistant-api run

down:
	docker compose -f deploy/docker-compose.yml down

request:
	curl --silent --show-error --max-time 600 \
		--header 'Accept: application/json' \
		--form 'audio=@$(AUDIO)' \
		--write-out '\nHTTP %{http_code}\n' \
		'$(API_URL)/v1/audiomessage'
