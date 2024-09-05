#!/usr/bin/env sh

go mod tidy
go build -o sway-yasm ./cmd/sway-yasm
