package main

import (
	"encoding/json"
	"github.com/larschri/blaner/dataset"
	"github.com/larschri/blaner/dataset/dtm10utm32"
	"github.com/larschri/blaner/render"
	"image/png"
	"log"
	"math"
	"net/http"
	_ "net/http/pprof"
	"os"
	"path/filepath"
	"runtime/pprof"
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
	easting, northing := dtm10utm32.DTM10UTM32Dataset.Translate(getFloatParam(req, "lat0"), getFloatParam(req, "lng0"))
	easting1, northing1 := dtm10utm32.DTM10UTM32Dataset.Translate(getFloatParam(req, "lat1"), getFloatParam(req, "lng1"))
	angle := -math.Atan2(easting-easting1, northing1-northing)
	return render.Renderer{
		Start:       angle - 0.05,
		Width:       .1,
		Columns:     800,
		Easting:     easting,
		Northing:    northing,
		HeightAngle: .16,
		MinHeight:   -.08,
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
	png.Encode(w, renderer.CreateImage())
}

func main() {
	files, err := filepath.Glob("dem-files/[^.]*.dem")
	if err != nil {
		panic(err)
	}

	elevmap, err = dataset.LoadFiles(&dtm10utm32.DTM10UTM32Dataset, files)
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
		img := render.Renderer{
			Start:       -2.2867,
			Width:       .1,
			Columns:     800,
			Easting:     463561,
			Northing:    6833871,
			HeightAngle: .16,
			MinHeight:   -.08,
			Elevations:  elevmap,
		}.CreateImage()
		file, _ := os.Create("foo.png")
		png.Encode(file, img)
	}
}
