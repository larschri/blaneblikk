package server

import (
	"encoding/json"
	"github.com/larschri/blaneblikk/dataset"
	"github.com/larschri/blaneblikk/dataset/dtm10utm32"
	"github.com/larschri/blaneblikk/render"
	"image/png"
	"log"
	"math"
	"net"
	"net/http"
	_ "net/http/pprof"
	"strconv"
)

type Server struct {
	ElevationMap dataset.ElevationMap
	HTTPServer http.Server
}

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

func (srv *Server) requestToRenderer(req *http.Request) render.Renderer {
	width := math.Pi * 2 / 64
	easting, northing := dtm10utm32.DTM10UTM32Dataset.Translate(getFloatParam(req, "lat0"), getFloatParam(req, "lng0"))
	easting1, northing1 := dtm10utm32.DTM10UTM32Dataset.Translate(getFloatParam(req, "lat1"), getFloatParam(req, "lng1"))
	angle := -math.Atan2(easting-easting1, northing1-northing)
	return render.Renderer{
		Start:      angle - width/2,
		Width:      width,
		Columns:    800,
		Easting:    easting,
		Northing:   northing,
		Elevations: srv.ElevationMap,
	}
}

func writeJSONResponse(w http.ResponseWriter, result interface{}, err error) {
	if err != nil {
		w.WriteHeader(400)
		_, err := w.Write([]byte(err.Error()))
		if err != nil {
			log.Printf("failed to write HTTP 400 response: %v", err)
		}
		return
	}

	bytes, err := json.Marshal(result)
	if err != nil {
		w.WriteHeader(500)
		_, err := w.Write([]byte(err.Error()))
		if err != nil {
			log.Printf("failed to write HTTP 500 response: %v", err)
		}
		return
	}

	w.Header().Add("Content-Type", "application/json")
	_, err = w.Write(bytes)
	if err != nil {
		panic(err)
	}
}

func (srv *Server) handlePixelToLatLng(w http.ResponseWriter, req *http.Request) {
	renderer := srv.requestToRenderer(req)

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

func (srv *Server) handleImageRequest(w http.ResponseWriter, req *http.Request) {
	w.Header().Add("Content-Type", "image/png")
	renderer := srv.requestToRenderer(req)
	err := (&png.Encoder{CompressionLevel: png.BestSpeed}).Encode(w, renderer.CreateImage())
	if err != nil {
		log.Printf("failed during image encoding: %v", err)
	}
}

func (srv *Server) Serve(hostPort string) error {
	listener, err := net.Listen("tcp", hostPort)
	if err != nil {
		return err
	}

	log.Print("Listening to " + hostPort)

	m := http.NewServeMux()
	m.HandleFunc("/bb/pixelLatLng", srv.handlePixelToLatLng)
	m.HandleFunc("/bb", srv.handleImageRequest)
	m.Handle("/", http.FileServer(http.Dir("server/static")))

	srv.HTTPServer.Handler = m
	return srv.HTTPServer.Serve(listener)
}
