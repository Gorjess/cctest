.PHONY:	all build run clean

BINARY=client

ifeq ($(OS),Windows_NT)
#	BINARY=client.exe
endif

build:
	@go build -o $(BINARY)
	@echo "finish building"

run:
	@./$(BINARY)

clean:
	@rm -f $(BINARY)
	@echo "finish cleaning"

