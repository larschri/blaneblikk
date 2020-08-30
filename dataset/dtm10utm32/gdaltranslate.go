package dtm10utm32

// #cgo CFLAGS: -Igdal
// #cgo LDFLAGS: -lgdal
// #include <ogr_srs_api.h>
// #include <stdlib.h>
import "C"

type DTM10UTM32 struct {
	trans C.OGRCoordinateTransformationH
	itrans C.OGRCoordinateTransformationH
	wkt string
}

var DTM10UTM32Dataset DTM10UTM32

func init() {
	DTM10UTM32Dataset.wkt = `PROJCS["UTM Zone 32, Northern Hemisphere",GEOGCS["WGS 84",DATUM["WGS_1984",SPHEROID["WGS 84",6378137,298.257223563,AUTHORITY["EPSG","7030"]],AUTHORITY["EPSG","6326"]],PRIMEM["Greenwich",0,AUTHORITY["EPSG","8901"]],UNIT["degree",0.0174532925199433,AUTHORITY["EPSG","9122"]],AUTHORITY["EPSG","4326"]],PROJECTION["Transverse_Mercator"],PARAMETER["latitude_of_origin",0],PARAMETER["central_meridian",9],PARAMETER["scale_factor",0.9996],PARAMETER["false_easting",500000],PARAMETER["false_northing",0],UNIT["Meter",1]]`
	UTM32WKT := C.CString(DTM10UTM32Dataset.wkt)
	UTM32SpatialReference := C.OSRNewSpatialReference(UTM32WKT)
	LatLngSpatialReference := C.OSRCloneGeogCS(UTM32SpatialReference)
	DTM10UTM32Dataset.trans = C.OCTNewCoordinateTransformation(LatLngSpatialReference, UTM32SpatialReference)
	DTM10UTM32Dataset.itrans = C.OCTNewCoordinateTransformation(UTM32SpatialReference, LatLngSpatialReference)
}

func (dtm *DTM10UTM32) Translate(lat float64, lng float64) (float64, float64) {
	xs := []float64{lng}
	ys := []float64{lat}
	zs := []float64{1}
	C.OCTTransform(dtm.trans, C.int(1), (*C.double)(&xs[0]), (*C.double)(&ys[0]), (*C.double)(&zs[0]))
	return xs[0], ys[0]
}

func (dtm *DTM10UTM32) ITranslate(easting float64, northing float64) (float64, float64) {
	xs := []float64{easting}
	ys := []float64{northing}
	zs := []float64{1}
	C.OCTTransform(dtm.itrans, C.int(1), (*C.double)(&xs[0]), (*C.double)(&ys[0]), (*C.double)(&zs[0]))
	return ys[0], xs[0]
}
