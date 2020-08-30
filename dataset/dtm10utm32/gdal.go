package dtm10utm32

// #cgo CFLAGS: -I../../gdal
// #cgo LDFLAGS: -lgdal
// #include <gdal.h>
// #include <stdlib.h>
import "C"
import (
	"log"
	"unsafe"
)


func init() {
	C.GDALAllRegister()
}

func (dtm *DTM10UTM32) ReadFile(fname string) (buffer [][]float32, minEasting float64, maxNorthing float64) {
	log.Printf("reading %s", fname)
	cstr := C.CString(fname)
	defer C.free(unsafe.Pointer(cstr))
	ds := C.GDALOpen(cstr, C.GA_ReadOnly)
	if ds == nil {
		panic("failed to read dem file")
	}

	wkt := C.GDALGetProjectionRef(ds)
	if C.GoString(wkt) != dtm.wkt {
		panic("Unexpected wkt for " + fname + ":" + C.GoString(wkt))
	}

	var gdalTransformArray [6]float64
	if C.GDALGetGeoTransform(ds, (*C.double)(&gdalTransformArray[0])) != C.CE_None {
		panic("failed to run transform")
	}
	if gdalTransformArray[1] != 10 || gdalTransformArray[5] != -10 {
		panic("unexpected file format")
	}

	xsize := C.GDALGetRasterXSize(ds)
	ysize := C.GDALGetRasterYSize(ds)
	if xsize < 5040 || xsize > 5050 || ysize < 5040 || ysize > 5050 {
		log.Panicf("unexpected dem file buffer size %d x %d", xsize, ysize)
		panic("")
	}

	buf := make([]float32, xsize*ysize)
	band := C.GDALGetRasterBand(ds, 1)
	if C.GDALRasterIO(band, C.GF_Read, 0, 0, xsize, ysize, unsafe.Pointer(&buf[0]), xsize, ysize, C.GDT_Float32, 0, 0) != C.CE_None {
		panic("failed to read elevation buffer from file")
	}

	// Input files are not aligned to the same global 10x10 matrix.
	// Offsets below are 205 for most files, but 210 and 215 for some
	eastingOffset := 1000 - int(gdalTransformArray[0])%1000
	northingOffset := int(gdalTransformArray[3]) % 1000
	colOffset := eastingOffset / 10
	rowOffset := northingOffset / 10

	minEasting = gdalTransformArray[0] + float64(eastingOffset)
	maxNorthing = gdalTransformArray[3] - float64(northingOffset)
	for i := 0; i < 5001; i++ {
		offset := colOffset + (i+rowOffset)*int(xsize)
		buffer = append(buffer, buf[offset:offset+5001])
	}
	return
}
