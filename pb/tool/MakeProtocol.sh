#!/bin/bash

./protoc -I=../raw/ --plugin=protoc-gen-go="./protoc-gen-go" --go_out=../ ../raw/*.proto

echo "proto compiled"
