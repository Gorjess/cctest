.PHONY:	all build install run dep clean echo

BINARY=chatserver
CHAT_ROOT=serverimpl/chat
TARGET_OS=
INSTALL_TOP=

ifeq ($(OS),Windows_NT)
    BINARY=$(BINARY).exe
    TARGET_OS=Windows
    INSTALL_TOP=D:/sevice
else
    TARGET_OS=$(shell uname)
    INSTALL_TOP=/usr/local/service
endif

build:
	@cd $(CHAT_ROOT) && go build -o ${BINARY} -ldflags "-X main.InstallAt=$(INSTALL_TOP)"
	@echo "finish building "$(BINARY)

install:
	@mkdir -p $(INSTALL_TOP)
	@cd $(CHAT_ROOT) && cp $(BINARY) conf/*.json $(INSTALL_TOP)
	@echo "finish installing"

run:
	@cd $(INSTALL_TOP) && ./$(BINARY)

dep:
	@go mod tidy

echo:
	@echo "install at: "$(INSTALL_TOP)
	@echo "server bin: "$(BINARY)

clean:
	@rm -rf $(INSTALL_TOP)/service
	@@cd $(CHAT_ROOT) && rm $(BINARY)
	@echo "finish cleaning"

