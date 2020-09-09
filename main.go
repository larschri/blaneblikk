package main

import (
	"encoding/json"
	"flag"
	"github.com/larschri/blaner/dataset"
	"github.com/larschri/blaner/dataset/dtm10utm32"
	"github.com/larschri/blaner/render"
	"image/png"
	"log"
	"math"
	"net"
	"net/http"
	_ "net/http/pprof"
	"path/filepath"
	"strconv"
)

var elevmap dataset.ElevationMap

func getFloatParam(req *http.Request, param string) float64 {
	f, err := strconv.ParseFloat(req.URL.Query().Get(param), 64)
	if err != nil {
		panic(err)
	}
	return f
}

func getIntParam(req *http.Request, param string) int {
	num, err := strconv.ParseInt(req.URL.Query().Get(param), 10, 32)
	if err != nil {
		panic(err)
	}
	return int(num)
}

func requestToRenderer(req *http.Request) render.Renderer {
	width := math.Pi * 2 / 64
	easting, northing := dtm10utm32.DTM10UTM32Dataset.Translate(getFloatParam(req, "lat0"), getFloatParam(req, "lng0"))
	easting1, northing1 := dtm10utm32.DTM10UTM32Dataset.Translate(getFloatParam(req, "lat1"), getFloatParam(req, "lng1"))
	angle := -math.Atan2(easting-easting1, northing1-northing)
	return render.Renderer{
		Start:       angle - width / 2,
		Width:       width,
		Columns:     800,
		Easting:     easting,
		Northing:    northing,
		Elevations:  elevmap,
	}
}

func writeJSONResponse(w http.ResponseWriter, result interface{}, err error) {
	if err != nil {
		w.WriteHeader(400)
		w.Write([]byte(err.Error()))
		return
	}

	bytes, err := json.Marshal(result)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
		return
	}

	w.Header().Add("Content-Type", "application/json")
	_, err = w.Write(bytes)
	if err != nil {
		panic(err)
	}
}

func pixelLatLngHandler(w http.ResponseWriter, req *http.Request) {
	renderer := requestToRenderer(req)
	pos, err := renderer.PixelToLatLng(getIntParam(req, "offsetX"), getIntParam(req, "offsetY"))
	if err != nil {
		writeJSONResponse(w, nil, err)
		return
	}
	lat, lng := dtm10utm32.DTM10UTM32Dataset.ITranslate(pos.Easting, pos.Northing)
	writeJSONResponse(w, map[string]interface{}{
		"lat": lat,
		"lng": lng,
	}, nil)
}

func blanerHandler(w http.ResponseWriter, req *http.Request) {
	w.Header().Add("Content-Type", "image/png")
	renderer := requestToRenderer(req)
	(&png.Encoder{CompressionLevel: png.BestSpeed}).Encode(w, renderer.CreateImage())
}

func main() {
	hostPort := flag.String("address", "localhost:8090", "http 'host:port' for the server")
	demFileDir := flag.String("demfiles", "dem-files", "directory with *.dem files")
	mmapFileDir := flag.String("mmapfiles", "/tmp", "directory for generated (optimised) *.mmap files")
	flag.Parse()
	files, err := filepath.Glob(*demFileDir + "/[^.]*.dem")
	if err != nil {
		panic(err)
	}

	elevmap, err = dataset.LoadFiles(&dtm10utm32.DTM10UTM32Dataset, *mmapFileDir, files)
	if err != nil {
		panic(err)
	}

	http.HandleFunc("/blaner/pixelLatLng", pixelLatLngHandler)
	http.HandleFunc("/blaner", blanerHandler)
	http.Handle("/", http.FileServer(http.Dir("htdocs")))

	listener, err := net.Listen("tcp", *hostPort)
	if err != nil {
		log.Fatal(err)
	}

	log.Print("Listening to " + *hostPort)

	_ = http.Serve(listener, nil)
}
