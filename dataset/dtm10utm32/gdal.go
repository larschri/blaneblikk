package dtm10utm32

// #cgo CFLAGS: -I../../gdal
// #cgo LDFLAGS: -lgdal
// #include <gdal.h>
// #include <stdlib.h>
import "C"
import (
	"errors"
	"log"
	"unsafe"
)

const UTM32WKT = `PROJCS["UTM Zone 32, Northern Hemisphere",GEOGCS["WGS 84",DATUM["WGS_1984",SPHEROID["WGS 84",6378137,298.257223563,AUTHORITY["EPSG","7030"]],AUTHORITY["EPSG","6326"]],PRIMEM["Greenwich",0,AUTHORITY["EPSG","8901"]],UNIT["degree",0.0174532925199433,AUTHORITY["EPSG","9122"]],AUTHORITY["EPSG","4326"]],PROJECTION["Transverse_Mercator"],PARAMETER["latitude_of_origin",0],PARAMETER["central_meridian",9],PARAMETER["scale_factor",0.9996],PARAMETER["false_easting",500000],PARAMETER["false_northing",0],UNIT["Meter",1]]`

func init() {
	C.GDALAllRegister()
}

func readGDAL(fname string) (error, Buffer5000) {
	log.Printf("reading %s", fname)
	cstr := C.CString(fname)
	defer C.free(unsafe.Pointer(cstr))
	ds := C.GDALOpen(cstr, C.GA_ReadOnly)
	if ds == nil {
		return errors.New("failed to read dem file"), Buffer5000{}
	}

	wkt := C.GDALGetProjectionRef(ds)
	if C.GoString(wkt) != UTM32WKT {
		panic("Unexpected wkt for " + fname + ":" + C.GoString(wkt))
	}

	var gdalTransformArray [6]float64
	if C.GDALGetGeoTransform(ds, (*C.double) (&gdalTransformArray[0])) != C.CE_None {
		return errors.New("failed to run transform"), Buffer5000{}
	}
	if gdalTransformArray[1] != 10 || gdalTransformArray[5] != -10 {
		return errors.New("unexpected file format"), Buffer5000{}
	}

	xsize := C.GDALGetRasterXSize(ds)
	ysize := C.GDALGetRasterYSize(ds)
	if xsize < 5040 || xsize > 5050 || ysize < 5040 || ysize > 5050 {
		log.Panicf("unexpected dem file buffer size %d x %d", xsize, ysize)
		panic("")
	}

	buf := make([]float32, xsize * ysize)
	band := C.GDALGetRasterBand(ds, 1)
	if C.GDALRasterIO(band, C.GF_Read, 0, 0, xsize, ysize, unsafe.Pointer(&buf[0]), xsize, ysize, C.GDT_Float32, 0, 0) != C.CE_None {
		return errors.New("failed to read elevation buffer from file"), Buffer5000{}
	}

	// Input files are not aligned to the same global 10x10 matrix.
	// Offsets below are 205 for most files, but 210 and 215 for some
	eastingOffset := 1000 - int(gdalTransformArray[0]) % 1000
	northingOffset := int(gdalTransformArray[3]) % 1000
	colOffset := eastingOffset / 10
	rowOffset := northingOffset / 10

	result := Buffer5000{
		EastingMin : gdalTransformArray[0] + float64(eastingOffset),
		NorthingMax : gdalTransformArray[3] - float64(northingOffset),
	}
	for i := 0; i < 5001; i++ {
		offset := colOffset + (i + rowOffset) * int(xsize)
		result.Buffer[i] = buf[offset:offset+5001]
	}
	return nil, result
}
