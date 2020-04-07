package main

// #cgo CFLAGS: -Igdal
// #cgo LDFLAGS: -lgdal
// #include <gdal.h>
import "C"

func main() {
	C.GDALAllRegister()
}
