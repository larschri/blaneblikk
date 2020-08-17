package dtm10utm32

// #cgo CFLAGS: -Igdal
// #cgo LDFLAGS: -lgdal
// #include <ogr_srs_api.h>
// #include <stdlib.h>
import "C"

var trans C.OGRCoordinateTransformationH
var itrans C.OGRCoordinateTransformationH

func init() {
	UTM32WKT := C.CString(UTM32WKT)
	UTM32SpatialReference := C.OSRNewSpatialReference(UTM32WKT)
	LatLngSpatialReference := C.OSRCloneGeogCS(UTM32SpatialReference)
	trans = C.OCTNewCoordinateTransformation(LatLngSpatialReference, UTM32SpatialReference)
	itrans = C.OCTNewCoordinateTransformation(UTM32SpatialReference, LatLngSpatialReference)
}

func Translate(lat float64, lng float64) (float64, float64) {
	xs := []float64{lng}
	ys := []float64{lat}
	zs := []float64{1}
	C.OCTTransform(trans, C.int(1), (*C.double)(&xs[0]), (*C.double)(&ys[0]), (*C.double)(&zs[0]))
	return xs[0], ys[0]
}

func ITranslate(easting float64, northing float64) (float64, float64) {
	xs := []float64{easting}
	ys := []float64{northing}
	zs := []float64{1}
	C.OCTTransform(itrans, C.int(1), (*C.double)(&xs[0]), (*C.double)(&ys[0]), (*C.double)(&zs[0]))
	return ys[0], xs[0]
}
