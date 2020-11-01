#!/bin/bash -eu

go test -run=. -bench=. -benchtime=5s -count 1 -benchmem -cpuprofile=cpu.out -memprofile=mem.out -trace=trace.out .
go tool pprof -http :8081 cpu.out
