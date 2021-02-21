#!/bin/bash

for ((i=0;i<10;i++)) do
	echo `go test -bench=. | grep Benchmark` >> benchmark_stat.txt
done
