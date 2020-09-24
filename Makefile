build:
	go build -o gotrace *.go && sudo ./gotrace google.com

.DEFAULT_GOAL := build