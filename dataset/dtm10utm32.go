package dataset

// #cgo CFLAGS: -I../gdal
// #cgo LDFLAGS: -lgdal
// #include <stdlib.h>
// #include <gdal.h>
// #include <ogr_srs_api.h>
import "C"

import (
	"log"
	"unsafe"
)

type DTM10UTM32 struct {
	trans  C.OGRCoordinateTransformationH
	itrans C.OGRCoordinateTransformationH
	wkt    string
}

var DTM10UTM32Dataset DTM10UTM32

func init() {
	C.GDALAllRegister()

	DTM10UTM32Dataset.wkt = `PROJCS["UTM Zone 32, Northern Hemisphere",GEOGCS["WGS 84",DATUM["WGS_1984",SPHEROID["WGS 84",6378137,298.257223563,AUTHORITY["EPSG","7030"]],AUTHORITY["EPSG","6326"]],PRIMEM["Greenwich",0,AUTHORITY["EPSG","8901"]],UNIT["degree",0.0174532925199433,AUTHORITY["EPSG","9122"]],AUTHORITY["EPSG","4326"]],PROJECTION["Transverse_Mercator"],PARAMETER["latitude_of_origin",0],PARAMETER["central_meridian",9],PARAMETER["scale_factor",0.9996],PARAMETER["false_easting",500000],PARAMETER["false_northing",0],UNIT["Meter",1]]`
	UTM32WKT := C.CString(DTM10UTM32Dataset.wkt)
	UTM32SpatialReference := C.OSRNewSpatialReference(UTM32WKT)
	LatLngSpatialReference := C.OSRCloneGeogCS(UTM32SpatialReference)
	DTM10UTM32Dataset.trans = C.OCTNewCoordinateTransformation(LatLngSpatialReference, UTM32SpatialReference)
	DTM10UTM32Dataset.itrans = C.OCTNewCoordinateTransformation(UTM32SpatialReference, LatLngSpatialReference)
}

// Translate translates lat/lng to easting/northing
func (dtm *DTM10UTM32) LatLngToUTM(lat float64, lng float64) (easting float64, northing float64) {
	xs := []float64{lng}
	ys := []float64{lat}
	zs := []float64{1}
	C.OCTTransform(dtm.trans, C.int(1), (*C.double)(&xs[0]), (*C.double)(&ys[0]), (*C.double)(&zs[0]))
	return xs[0], ys[0]
}

// ITranslate translates easting/northing to lat/lng
func (dtm *DTM10UTM32) UTMToLatLng(easting float64, northing float64) (lat float64, lng float64) {
	xs := []float64{easting}
	ys := []float64{northing}
	zs := []float64{1}
	C.OCTTransform(dtm.itrans, C.int(1), (*C.double)(&xs[0]), (*C.double)(&ys[0]), (*C.double)(&zs[0]))
	return ys[0], xs[0]
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
