package main

import (
	"encoding/json"
	"github.com/larschri/blaner/dataset/dataset5000"
	"github.com/larschri/blaner/dataset/dtm10utm32"
	"github.com/larschri/blaner/render"
	"image/png"
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"runtime/pprof"
	"strconv"
)

var elevmap dataset5000.ElevationMap

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

func requestToRenderArgs(req *http.Request) render.Args {
	easting, northing := dtm10utm32.Translate(getFloatParam(req, "lat0"), getFloatParam(req, "lng0"))
	easting1, northing1 := dtm10utm32.Translate(getFloatParam(req, "lat1"), getFloatParam(req, "lng1"))
	angle := -math.Atan2(easting-easting1, northing1-northing)
	return render.Args{
		Start:       angle - 0.05,
		Width:       .1,
		Columns:     800,
		Easting:     easting,
		Northing:    northing,
		HeightAngle: .16,
		MinHeight:   -.08,
	}
}

func writeResponse(w http.ResponseWriter, result interface{}, err error) {
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

	_, err = w.Write(bytes)
	if err != nil {
		panic(err)
	}
}

func pixelLatLngHandler(w http.ResponseWriter, req *http.Request) {
	w.Header().Add("Content-Type", "application/json")
	xx := requestToRenderArgs(req)
	latlng, err := render.PixelToLatLng(xx, elevmap, getIntParam(req, "offsetX"), getIntParam(req, "offsetY"))
	writeResponse(w, latlng, err)
}

func blanerHandler(w http.ResponseWriter, req *http.Request) {
	w.Header().Add("Content-Type", "image/png")
	xx := requestToRenderArgs(req)
	png.Encode(w, render.CreateImage(xx, elevmap))
}

func main() {
	files, err := filepath.Glob("dem-files/[^.]*.dem")
	if err != nil {
		panic(err)
	}
	elevmap, err = dataset5000.LoadFiles(dtm10utm32.Dataset{}, files)
	if err != nil {
		panic(err)
	}

	if len(os.Args) < 2 {
		http.HandleFunc("/blaner/pixelLatLng", pixelLatLngHandler)
		http.HandleFunc("/blaner", blanerHandler)
		http.Handle("/", http.FileServer(http.Dir("htdocs")))
		err := http.ListenAndServe(":8090", nil)
		if err != nil {
			panic(err)
		}
	} else {
		f, err := os.Create("cpuprof")
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
		img := render.CreateImage(render.Args{
			Start:       -2.2867,
			Width:       .1,
			Columns:     400,
			Easting:     463561,
			Northing:    6833871,
			HeightAngle: .16,
			MinHeight:   -.08,
		}, elevmap)
		file, _ := os.Create("foo.png")
		png.Encode(file, img)
	}
}
