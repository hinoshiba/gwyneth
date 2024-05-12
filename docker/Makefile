.PHONY: all
all: build up

.PHONY: build
build:
	make -C ../ d-build
	mkdir -p ./dockerfiles/gwyneth/bin/
	cp ../bin/gwyneth ./dockerfiles/gwyneth/bin/gwyneth
	rm -rf ./dockerfiles/gwyneth/http
	cp -rf ../http ./dockerfiles/gwyneth/http
	rm -rf ./dockerfiles/gwyneth/scripts
	cp -rf ../scripts ./dockerfiles/gwyneth/scripts
	docker compose build

.PHONY: up
up:
	docker compose up

.PHONY: down
down:
	docker compose down

.PHONY:clean
clean:
	rm -rf ./dockerfiles/gwyneth/bin/
	rm -rf ./dockerfiles/gwyneth/http
	rm -rf ./var/mysql
	docker compose down
