@echo off

protoc -I=../raw/ --plugin=protoc-gen-go=%GOBIN%"/protoc-gen-go.exe" --go_out=../ ../raw/*.proto

echo "proto compiled"

pause