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
	cStr := C.CString(fname)
	defer C.free(unsafe.Pointer(cStr))
	ds := C.GDALOpen(cStr, C.GA_ReadOnly)
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

	xSize := C.GDALGetRasterXSize(ds)
	ySize := C.GDALGetRasterYSize(ds)
	if xSize < 5040 || xSize > 5050 || ySize < 5040 || ySize > 5050 {
		log.Panicf("unexpected dem file buffer size %d x %d", xSize, ySize)
		panic("")
	}

	buf := make([]float32, xSize*ySize)
	band := C.GDALGetRasterBand(ds, 1)
	if C.GDALRasterIO(band, C.GF_Read, 0, 0, xSize, ySize, unsafe.Pointer(&buf[0]), xSize, ySize, C.GDT_Float32, 0, 0) != C.CE_None {
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
		offset := colOffset + (i+rowOffset)*int(xSize)
		buffer = append(buffer, buf[offset:offset+5001])
	}
	return
}
