package server

import (
	"context"
	"encoding/json"
	"fmt"
	"image/png"
	"log"
	"math"
	"net"
	"net/http"
	_ "net/http/pprof"
	"strconv"
	"time"

	"github.com/larschri/blaneblikk/dataset"
	"github.com/larschri/blaneblikk/render"
)

type Server struct {
	ElevationMap dataset.ElevationMap
	Listener     net.Listener
}

func (srv *Server) requestToRenderer(req *http.Request) (render.Renderer, error) {
	lat0, err := strconv.ParseFloat(req.URL.Query().Get("lat0"), 64)
	if err != nil {
		return render.Renderer{}, fmt.Errorf("failed to parse lat0")
	}

	lng0, err := strconv.ParseFloat(req.URL.Query().Get("lng0"), 64)
	if err != nil {
		return render.Renderer{}, fmt.Errorf("failed to parse lng0")
	}

	lat1, err := strconv.ParseFloat(req.URL.Query().Get("lat1"), 64)
	if err != nil {
		return render.Renderer{}, fmt.Errorf("failed to parse lat1")
	}

	lng1, err := strconv.ParseFloat(req.URL.Query().Get("lng1"), 64)
	if err != nil {
		return render.Renderer{}, fmt.Errorf("failed to parse lng1")
	}

	easting, northing := dataset.DTM10UTM32Dataset.LatLngToUTM(lat0, lng0)
	easting1, northing1 := dataset.DTM10UTM32Dataset.LatLngToUTM(lat1, lng1)

	width := math.Pi * 2 / 64
	angle := -math.Atan2(easting-easting1, northing1-northing)

	return render.Renderer{
		Start:      angle - width/2,
		Width:      width,
		Columns:    800,
		Easting:    easting,
		Northing:   northing,
		Elevations: srv.ElevationMap,
	}, nil
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
		log.Printf("failed to write HTTP response: %v", err)
	}
}

func (srv *Server) handlePixelToLatLng(w http.ResponseWriter, req *http.Request) {
	renderer, err := srv.requestToRenderer(req)
	if err != nil {
		writeJSONResponse(w, nil, err)
		return
	}

	offsetX, err := strconv.ParseInt(req.URL.Query().Get("offsetX"), 10, 32)
	if err != nil {
		writeJSONResponse(w, nil, fmt.Errorf("failed to parse 'offsetX': %w", err))
		return
	}

	offsetY, err := strconv.ParseInt(req.URL.Query().Get("offsetY"), 10, 32)
	if err != nil {
		writeJSONResponse(w, nil, fmt.Errorf("failed to parse 'offsetY': %w", err))
		return
	}

	easting, northing, err := renderer.PixelToUTM(int(offsetX), int(offsetY))
	if err != nil {
		writeJSONResponse(w, nil, err)
		return
	}

	lat, lng := dataset.DTM10UTM32Dataset.UTMToLatLng(easting, northing)
	writeJSONResponse(w, map[string]interface{}{
		"lat": lat,
		"lng": lng,
	}, nil)
}

func (srv *Server) handleImageRequest(w http.ResponseWriter, req *http.Request) {
	w.Header().Add("Content-Type", "image/png")

	renderer, err := srv.requestToRenderer(req)
	if err != nil {
		writeJSONResponse(w, nil, err)
		return
	}

	err = (&png.Encoder{CompressionLevel: png.BestSpeed}).Encode(w, renderer.CreateImage())
	if err != nil {
		log.Printf("failed during image encoding: %v", err)
	}
}

// shutdownWhenDone invokes http.Server.Shutdown when the given context is cancelled.
// This function will block until context cancellation.
func shutdownWhenDone(ctx context.Context, server *http.Server) {
	log.Print("server started")
	<-ctx.Done()

	c, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	log.Print("teminating server")
	server.Shutdown(c)
}

func (srv *Server) Serve(ctx context.Context) error {
	m := http.NewServeMux()
	m.HandleFunc("/bb/pixelLatLng", srv.handlePixelToLatLng)
	m.HandleFunc("/bb", srv.handleImageRequest)
	m.Handle("/", http.FileServer(http.Dir("server/static")))

	server := http.Server{
		Handler: m,
	}

	go shutdownWhenDone(ctx, &server)

	err := server.Serve(srv.Listener)

	if ctx.Done() == nil {
		return err
	}

	log.Print("server stopped")
	return nil
}
