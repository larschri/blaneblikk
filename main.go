package main

import (
	"flag"
	"github.com/larschri/blaneblikk/dataset"
	"github.com/larschri/blaneblikk/dataset/dtm10utm32"
	"github.com/larschri/blaneblikk/server"
	_ "net/http/pprof"
	"path/filepath"
)

func main() {
	hostPort := flag.String("address", "localhost:8090", "http 'host:port' for the server")
	demFileDir := flag.String("demfiles", "dem-files", "directory with *.dem files")
	mmapFileDir := flag.String("mmapfiles", "/tmp", "directory for generated (optimised) *.mmap files")
	flag.Parse()
	files, err := filepath.Glob(*demFileDir + "/[^.]*.dem")
	if err != nil {
		panic(err)
	}

	elevationMap, err := dataset.LoadFiles(&dtm10utm32.DTM10UTM32Dataset, *mmapFileDir, files)
	if err != nil {
		panic(err)
	}

	s := server.Server{
		ElevationMap: elevationMap,
	}

	if err = s.Serve(*hostPort); err != nil {
		panic(err)
	}
}
