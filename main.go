package main

import (
	"fmt"
	"github.com/larschri/blaner/dataset/dtm10utm32"
	"github.com/larschri/blaner/render"
	"image/png"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
)

var args1 = render.Args{
	Start:    3.1,
	Width:    .1,
	Columns:  400,
	Step:     10,
	Easting:  463561,
	Northing: 6833871,
	HeightAngle: .16,
	MinHeight:   -.08,
}

var elevmap dtm10utm32.ElevationMap

func getFloatParam(req *http.Request, param string) float64 {
	f, err := strconv.ParseFloat(req.URL.Query().Get(param), 64)
	if err != nil {
		panic(err)
	}
	return f
}

func blanerHandler(w http.ResponseWriter, req *http.Request) {
	w.Header().Add("Content-Type", "image/png")
	easting, northing := Translate(getFloatParam(req, "lat0"), getFloatParam(req, "lng0"))
	easting1, northing1 := Translate(getFloatParam(req, "lat1"), getFloatParam(req, "lng1"))
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
	elevmap, err = dtm10utm32.LoadFiles(files)
	if err != nil {
		panic(err)
	}

	if len(os.Args) < 2 {
		http.HandleFunc("/blaner", blanerHandler)
		http.Handle("/", http.FileServer(http.Dir("htdocs")))
		http.ListenAndServe(":8090", nil)
	} else {
		img := render.CreateImage(args1, elevmap)
		f, _ := os.Create("foo.png")
		png.Encode(f, img)
	}
}
