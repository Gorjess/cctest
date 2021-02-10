#!/bin/bash

#clean
#rm -rf *.pb.go



#build
cd proto
protoc -I=./ --plugin=protoc-gen-go="../protoc-gen-go" --go_out=../ ./*.proto
#install
#cp csprotocol.pb.go ssprotocol.pb.go ../gameserver/msg/
#cp ssprotocol.pb.go ../matchserver/msg/
