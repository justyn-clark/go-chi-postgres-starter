set shell := ["bash", "-eu", "-o", "pipefail", "-c"]

default:
    @just --list

help:
    make help

swagger:
    make swagger

swagger-serve:
    make swagger-serve

build:
    make build

run:
    make run

run-dev:
    make run-dev

dev:
    make dev

stop:
    make stop

lint:
    make lint

lint-fix:
    make lint-fix

fmt:
    make fmt

vet:
    make vet

test:
    make test

test-coverage:
    make test-coverage

test-utils:
    make test-utils

bench-utils:
    make bench-utils

test-setup:
    make test-setup

check:
    make check

migrate-up:
    make migrate-up

migrate-down:
    make migrate-down

migrate-create name="changes":
    NAME='{{name}}' make migrate-create

migrate-status:
    make migrate-status

docker-up:
    make docker-up

docker-down:
    make docker-down

docker-build:
    make docker-build

clean:
    make clean
