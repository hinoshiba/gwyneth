.PHONY: all
all: build up

.PHONY: build
build:
	make -C ../ d-build
	mkdir -p ./dockerfiles/gwyneth/bin/
	cp ../bin/gwyneth ./dockerfiles/gwyneth/bin/gwyneth
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
	rm -rf ./var/mysql
	docker compose down
