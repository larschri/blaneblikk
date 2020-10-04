package dataset

import (
	"io/ioutil"
	"math"
	"os"
	"path"
	"syscall"
	"unsafe"
)

// Elevation16 contains elevation values stored as 1/10 meter
type Elevation16 int16

// Elevation16Unit should be used to convert Elevation16 to meter
const Elevation16Unit = 0.1

// mmap5000 contains elevation data stored on disk and loaded into memory using mmap.
type mmap5000 struct {
	EastingMin    float64
	NorthingMax   float64
	MaxElevations [numberOfElevationMaplets][numberOfElevationMaplets]Elevation16

	// Elevations is a matrix of ElevationMaplets
	Elevations [numberOfElevationMaplets][numberOfElevationMaplets]ElevationMaplet
}

// DatasetReader reads elevation data from a file and returns it as a matrix
type DatasetReader interface {
	// ReadFile reads elevation data from a file and returns it as a matrix
	ReadFile(fname string) (buffer [][]float32, minEasting float64, maxNorthing float64)
}

const mmapStructSize = unsafe.Sizeof(mmap5000{})

// Close does nothing today
func (m *mmap5000) Close() error {
	return nil
}

func toMmapStruct(buf [][]float32) *mmap5000 {

	result := mmap5000{}

	for i := 0; i < numberOfElevationMaplets; i++ {
		for j := 0; j < numberOfElevationMaplets; j++ {
			// The loops below _includes_ the 25th element to compute MaxElevations.
			// Otherwise there would be a 10 meter gap between each 25x25 matrix.
			for m := 0; m <= ElevationMapletSize; m++ {
				row := i*ElevationMapletSize + m
				for n := 0; n <= ElevationMapletSize; n++ {
					col := j*ElevationMapletSize + n

					floatVal := buf[row][col]
					intVal := Elevation16(math.Round(float64(floatVal) / Elevation16Unit))

					if m < ElevationMapletSize && n < ElevationMapletSize {
						result.Elevations[i][j][m][n] = intVal
					}

					if intVal > result.MaxElevations[i][j] {
						result.MaxElevations[i][j] = intVal
					}
				}
			}
		}
	}
	return &result
}

// loadAsMmap will load the given fname using syscall.mmap
// The data can be accessed through the returned *mmap5000.
// The returned *os.File should be syscall.munmapped to release the resource.
func loadAsMmap(datasetReader DatasetReader, mmapFileDir string, fname string) (*mmap5000, error) {
	mmapFName := mmapFileDir + "/" + path.Base(fname) + ".mmap"
	fileInfo, err := os.Stat(fname)
	if err != nil {
		return nil, err
	}

	mmapFileInfo, err := os.Stat(mmapFName)
	if err != nil || fileInfo.ModTime().After(mmapFileInfo.ModTime()) || mmapFileInfo.Size() != int64(mmapStructSize) {
		err = writeMmapped(datasetReader, fname, mmapFName)
		if err != nil {
			return nil, err
		}
	}

	return openMmapped(mmapFName)
}

func writeMmapped(datasetReader DatasetReader, fname string, mmapFname string) error {
	buf, e, n := datasetReader.ReadFile(fname)
	mmapData := toMmapStruct(buf)
	mmapData.EastingMin = e
	mmapData.NorthingMax = n

	var bytes = (*(*[mmapStructSize]byte)(unsafe.Pointer(mmapData)))[:]
	return ioutil.WriteFile(mmapFname, bytes, 0644)
}

func openMmapped(fname string) (*mmap5000, error) {
	file, err := os.OpenFile(fname, os.O_RDONLY, 0)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	data, err := syscall.Mmap(int(file.Fd()), 0, int(mmapStructSize), syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil {
		return nil, err
	}
	return (*mmap5000)(unsafe.Pointer(&data[0])), nil
}
