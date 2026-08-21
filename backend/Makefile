.PHONY: test race vet build run seed-user

test:
	$(MAKE) -C .. test

race:
	$(MAKE) -C .. race

vet:
	$(MAKE) -C .. vet

build:
	$(MAKE) -C .. build

run:
	cd .. && go run ./cmd/server

seed-user:
	cd .. && go run ./cmd/seed-user
