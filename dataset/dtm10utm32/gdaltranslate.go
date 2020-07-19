package dtm10utm32

// #cgo CFLAGS: -Igdal
// #cgo LDFLAGS: -lgdal
// #include <ogr_srs_api.h>
// #include <stdlib.h>
import "C"

var trans C.OGRCoordinateTransformationH

func init() {
	UTM32WKT := C.CString(UTM32WKT)
	UTM32SpatialReference := C.OSRNewSpatialReference(UTM32WKT)
	LatLngSpatialReference := C.OSRCloneGeogCS(UTM32SpatialReference)
	trans = C.OCTNewCoordinateTransformation(LatLngSpatialReference, UTM32SpatialReference)
}

func Translate(lat float64, lng float64) (float64, float64) {
	xs := []float64{lng}
	ys := []float64{lat}
	zs := []float64{1}
	C.OCTTransform(trans, C.int(1), (*C.double)(&xs[0]), (*C.double)(&ys[0]), (*C.double)(&zs[0]))
	return xs[0], ys[0]
}
