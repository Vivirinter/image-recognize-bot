help : Makefile
	@echo "\nAVAILABLE COMMANDS\n"
	@cat Makefile | grep "##" | sed -n 's/^## /make /p' | column -t -s ':' && echo ""

.DEFAULT_GOAL := help


recognize_build:
	docker build -t recognize .


recognize_run:
	docker run -it -p 8080:8080 recognize


recognize_build_run:
	make recognition_build && make recognize


bot_run:
	go run cmd/bot/main.go