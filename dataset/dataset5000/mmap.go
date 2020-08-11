package dataset5000

import (
	"io/ioutil"
	"math"
	"os"
	"path"
	"syscall"
	"unsafe"
)

const numberOfSmallSquares = bigSquareSize / smallSquareSize

// elvation16 contains elevation values stored as 1/10 meter
type elevation16 int16

// elevation16Multiplier should be used to convert elevation16 to meter
const elevation16Unit = 0.1

// Mmap5000 contains elevation data stored on disk and loaded into memory using mmap.
type Mmap5000 struct {
	EastingMin    float64
	NorthingMax   float64
	MaxElevations [numberOfSmallSquares][numberOfSmallSquares]elevation16

	// Elvations is a matrix of elevation matrices.
	// An elevation data point is a elevation16 where a unit corresponds to 0.1 meter of elevation
	// An elevation matrix contains 25x25 such elevation data points
	// This matrix contains 200x200 such elevation matrices.
	Elevations [numberOfSmallSquares][numberOfSmallSquares][smallSquareSize][smallSquareSize]elevation16
}

type DatasetReader interface {
	ReadFile(fname string) (buffer [][]float32, minEasting float64, maxNorthing float64)
}

const mmapstructSize = unsafe.Sizeof(Mmap5000{})

func (m *Mmap5000) Close() error {
	return nil
}

func toMmapStruct(buf [][]float32) *Mmap5000 {

	result := Mmap5000{}

	for i := 0; i < numberOfSmallSquares; i++ {
		for j := 0; j < numberOfSmallSquares; j++ {
			// The loops below _includes_ the 25th element to compute MaxElevations.
			// Otherwise there would be a 10 meter gap between each 25x25 matrix.
			for m := 0; m <= smallSquareSize; m++ {
				row := i*smallSquareSize + m
				for n := 0; n <= smallSquareSize; n++ {
					col := j*smallSquareSize + n

					floatval := buf[row][col]
					intval := elevation16(math.Round(float64(floatval) / elevation16Unit))

					if m < smallSquareSize && n < smallSquareSize {
						result.Elevations[i][j][m][n] = intval
					}

					if intval > result.MaxElevations[i][j] {
						result.MaxElevations[i][j] = intval
					}
				}
			}
		}
	}
	return &result
}

// LoadAsMmap will load the given fname using syscall.mmap
// The data can be accessed through the returned *Mmap5000.
// The returned *os.File should be syscall.munmapped to release the resource.
func LoadAsMmap(datasetReader DatasetReader, fname string) (*Mmap5000, error) {
	mmapFname := "/tmp/" + path.Base(fname) + ".mmap"
	fileInfo, err := os.Stat(fname)
	if err != nil {
		return nil, err
	}

	mmapFileInfo, err := os.Stat(mmapFname)
	if err != nil || fileInfo.ModTime().After(mmapFileInfo.ModTime()) || mmapFileInfo.Size() != int64(mmapstructSize) {
		err = writeMmapped(datasetReader, fname, mmapFname)
		if err != nil {
			return nil, err
		}
	}

	return openMmapped(mmapFname)
}

func writeMmapped(datasetReader DatasetReader, fname string, mmapFname string) error {
	buf, e, n := datasetReader.ReadFile(fname)
	mmapdata := toMmapStruct(buf)
	mmapdata.EastingMin = e
	mmapdata.NorthingMax = n

	var bytes = (*(*[mmapstructSize]byte)(unsafe.Pointer(mmapdata)))[:]
	return ioutil.WriteFile(mmapFname, bytes, 0644)
}

func openMmapped(fname string) (*Mmap5000, error) {
	file, err := os.OpenFile(fname, os.O_RDONLY, 0)
	defer file.Close()

	if err != nil {
		return nil, err
	}

	data, err := syscall.Mmap(int(file.Fd()), 0, int(mmapstructSize), syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil {
		return nil, err
	}
	return (*Mmap5000)(unsafe.Pointer(&data[0])), nil
}
