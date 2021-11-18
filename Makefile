.PHONY: all up down logs test

all: rebuild refresh test

reboot: down up

rebuild: down build up

test: usa-test kgs-test

refresh: usa-refresh kgs-refresh

up:
	docker-compose up -d

down:
	docker-compose down --remove-orphans

build:
	docker-compose build

usa-test:
	docker-compose run --entrypoint="go test -coverprofile cover.out ./" url-shorten-app

kgs-test:
	docker-compose run --entrypoint="go test -coverprofile cover.out ./" key-gen-svc

usa-refresh:
	docker-compose run --entrypoint="/main --refresh-index" url-shorten-app

kgs-refresh:
	docker-compose run --entrypoint="/main --refresh-database" key-gen-svc

logs:
	docker-compose logs --tail=20 -f url-shorten-app key-gen-svc
