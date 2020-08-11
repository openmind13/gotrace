build:
	go build -o gotrace *.go && sudo ./gotrace

.DEFAULT_GOAL := build