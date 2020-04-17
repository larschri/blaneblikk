package elevationmap

// #cgo CFLAGS: -I../gdal
// #cgo LDFLAGS: -lgdal
// #include <gdal.h>
// #include <stdlib.h>
import "C"
import (
	"errors"
	"unsafe"
)

type gdalbuffer struct {
	eastingMin float64
	northingMax float64
	buffer []float32
}

func init() {
	C.GDALAllRegister()
}

func readGDAL(fname string) (error, gdalbuffer) {
	cstr := C.CString(fname)
	defer C.free(unsafe.Pointer(cstr))
	ds := C.GDALOpen(cstr, C.GA_ReadOnly)
	if ds == nil {
		return errors.New("failed to read dem file"), gdalbuffer{}
	}

	var gdalTransformArray [6]float64
	if C.GDALGetGeoTransform(ds, (*C.double) (&gdalTransformArray[0])) != C.CE_None {
		return errors.New("failed to run transform"), gdalbuffer{}
	}
	if gdalTransformArray[1] != 10 || gdalTransformArray[5] != -10 {
		return errors.New("unexpected file format"), gdalbuffer{}
	}

	xsize := C.GDALGetRasterXSize(ds)
	ysize := C.GDALGetRasterYSize(ds)
	buf := make([]float32, xsize * ysize)
	band := C.GDALGetRasterBand(ds, 1)
	if C.GDALRasterIO(band, C.GF_Read, 0, 0, xsize, ysize, unsafe.Pointer(&buf[0]), xsize, ysize, C.GDT_Float32, 0, 0) != C.CE_None {
		return errors.New("failed to read elevation buffer from file"), gdalbuffer{}
	}

	return nil, gdalbuffer{
		eastingMin:  gdalTransformArray[0],
		northingMax: gdalTransformArray[3],
		buffer:      buf,
	}
}
