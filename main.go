package main

import (
	"context"
	"flag"
	"log"
	"net"
	"os"
	"os/signal"
	"path/filepath"

	"github.com/larschri/blaneblikk/dataset"
	"github.com/larschri/blaneblikk/server"
)

func newServer(demFileDir, mmapFileDir, hostPort string) (*server.Server, error) {
	files, err := filepath.Glob(demFileDir + "/[^.]*.dem")
	if err != nil {
		return nil, err
	}

	elevationMap, err := dataset.LoadFiles(&dataset.DTM10UTM32Dataset, mmapFileDir, files)
	if err != nil {
		return nil, err
	}

	listener, err := net.Listen("tcp", hostPort)
	if err != nil {
		return nil, err
	}
	log.Print("Listening to " + listener.Addr().String())

	return &server.Server{
		ElevationMap: elevationMap,
		Listener:     listener,
	}, nil

}

// withCancelOnSignal create a context that is cancelled when the process is interrupted.
// The parent parameter is used as the parent context.
func withCancelOnInterrupt(parent context.Context) context.Context {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	ctx, cancel := context.WithCancel(parent)

	go func() {
		<-c
		cancel()
	}()

	return ctx
}

func main() {
	hostPort := flag.String("address", "localhost:8090", "http 'host:port' for the server")
	demFileDir := flag.String("demfiles", "dem-files", "directory with *.dem files")
	mmapFileDir := flag.String("mmapfiles", "/tmp", "directory for generated (optimised) *.mmap files")
	flag.Parse()

	s, err := newServer(*demFileDir, *mmapFileDir, *hostPort)
	if err != nil {
		panic(err)
	}

	ctx := withCancelOnInterrupt(context.Background())

	if err = s.Serve(ctx); err != nil {
		panic(err)
	}
}
