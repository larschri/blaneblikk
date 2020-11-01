package main

import (
	"net"
	"net/http"
	"os"
	"testing"
)

var addr net.Addr

func TestMain(m *testing.M) {
	s, err := newServer("dem-files", "/tmp", ":0")
	if err != nil {
		panic(err)
	}

	addr = s.Listener.Addr()

	go func() {
		if err = s.Serve(); err != nil {
			panic(err)
		}
	}()

	os.Exit(m.Run())
}

func BenchmarkGaldhopiggen(b *testing.B) {
	for i := 0; i < b.N; i++ {
		resp, err := http.Get("http://" + addr.String() + "/bb?lat0=60.16542574699484&lng0=10.393753051757814&lat1=59.97039127513498&lng1=10.156130790710451")
		if err != nil {
			b.Fatal(err)
		}

		if resp.StatusCode != 200 {
			b.Fatal(resp.Status)
		}
	}
}
