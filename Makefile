.PHONY:	all build install run dep clean echo

BINARY=chatserver
CHAT_ROOT=serverimpl/chat
TARGET_OS=
INSTALL_TOP=

ifeq ($(OS),Windows_NT)
    TARGET_OS=Windows
    INSTALL_TOP=../chat
else
    TARGET_OS=$(shell uname)
    INSTALL_TOP=/usr/local/chatservice
endif

build:
	@cd $(CHAT_ROOT) && go build -o ${BINARY} -ldflags "-X main.InstallAt=$(INSTALL_TOP)"
	@echo "finish building "$(BINARY)

install:
	@test -d $(INSTALL_TOP) && rm -rf $(INSTALL_TOP)
	@rm -rf $(INSTALL_TOP)
	@mkdir -p $(INSTALL_TOP)/conf
	@cp $(CHAT_ROOT)/$(BINARY) $(INSTALL_TOP)
	@cp $(CHAT_ROOT)/list.txt $(INSTALL_TOP)
	@cp $(CHAT_ROOT)/conf/config.json $(INSTALL_TOP)/conf
	@echo "finish installing"

run:
	@cd $(INSTALL_TOP) && ./$(BINARY)

dep:
	@go mod tidy

echo:
	@echo "install at: "$(INSTALL_TOP)
	@echo "server bin: "$(BINARY)

clean:
	@rm -rf $(INSTALL_TOP)
	@@cd $(CHAT_ROOT) && rm $(BINARY)
	@echo "finish cleaning"

