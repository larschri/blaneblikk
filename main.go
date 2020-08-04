package main

import (
	"fmt"
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

var args1 = render.Args{
	Start:    -2.2867,
	Width:    .1,
	Columns:  400,
	Step:     10,
	Easting:  463561,
	Northing: 6833871,
	HeightAngle: .16,
	MinHeight:   -.08,
}

/*var args1 = render.Args{
	Start:    -2.2867,
	Width:    .1,
	Columns:  400,
	Step:     10,
	Easting:  463564,
	Northing: 6833871,
	HeightAngle: .16,
	MinHeight:   -.08,
}*/

var elevmap dataset5000.ElevationMap

func getFloatParam(req *http.Request, param string) float64 {
	f, err := strconv.ParseFloat(req.URL.Query().Get(param), 64)
	if err != nil {
		panic(err)
	}
	return f
}

func blanerHandler(w http.ResponseWriter, req *http.Request) {
	w.Header().Add("Content-Type", "image/png")
	easting, northing := dtm10utm32.Translate(getFloatParam(req, "lat0"), getFloatParam(req, "lng0"))
	easting1, northing1 := dtm10utm32.Translate(getFloatParam(req, "lat1"), getFloatParam(req, "lng1"))
	angle := -math.Atan2(easting-easting1, northing1-northing)
	fmt.Println(angle)
	xx := render.Args{
		Start:       angle - 0.05,
		Width:       .1,
		Columns:     400,
		Step:        10,
		Easting:     easting,
		Northing:    northing,
		HeightAngle: .16,
		MinHeight:   -.08,
	}
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
		img := render.CreateImage(args1, elevmap)
		file, _ := os.Create("foo.png")
		png.Encode(file, img)
	}
}
