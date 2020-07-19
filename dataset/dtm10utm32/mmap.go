package dtm10utm32

import (
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"path"
	"syscall"
	"unsafe"
)

// Mmapstruct contains elevation data stored on disk and loaded into memory using mmap.
type Mmapstruct struct {
	EastingMin float64
	NorthingMax float64
	MaxElevations [200][200]int16

	// Elvations is a matrix of elevation matrices.
	// An elevation data point is a int16 where a unit corresponds to 0.1 meter of elevation
	// An elevation matrix contains 25x25 such elevation data points
	// This matrix contains 200x200 such elevation matrices.
	Elevations [200][200][25][25]int16
}

const mmapstructSize = unsafe.Sizeof(Mmapstruct{})

func (m *Mmapstruct) Close() error {
	return nil
}

func toMmapStruct(buf gdalbuffer) *Mmapstruct {
	// Input files are not aligned to the same global 10x10 matrix.
	// Offsets below are 205 for most files, but 210 and 215 for some
	eastingOffset := 1000 - int(buf.eastingMin) % 1000
	northingOffset := int(buf.northingMax) % 1000
	result := Mmapstruct{
		EastingMin: buf.eastingMin + float64(eastingOffset),
		NorthingMax: buf.northingMax - float64(northingOffset),
	}
	colOffset := eastingOffset / 10
	rowOffset := northingOffset / 10

	for i := 0; i < 200; i++ {
		for j := 0; j < 200; j++ {
			// The loops below _includes_ the 25th element to compute MaxElevations.
			// Otherwise there would be a 10 meter gap between each 25x25 matrix.
			for m := 0; m <= 25; m++ {
				row := rowOffset + i * 25 + m
				for n := 0; n <= 25; n++ {
					col := colOffset + j * 25 + n

					floatval := buf.buffer[row * buf.xsize + col]
					intval := int16(math.Round(10 * float64(floatval)))

					if m < 25 && n < 25 {
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
// The data can be accessed through the returned *Mmapstruct.
// The returned *os.File should be syscall.munmapped to release the resource.
func LoadAsMmap(fname string) (*Mmapstruct, error) {
	mmapFname := "/tmp/" + path.Base(fname) + ".mmap"
	fileInfo, err := os.Stat(fname)
	if err != nil {
		return nil, err
	}

	mmapFileInfo, err := os.Stat(mmapFname)
	if err != nil || fileInfo.ModTime().After(mmapFileInfo.ModTime()) || mmapFileInfo.Size() != int64(mmapstructSize) {
		err = writeMmapped(fname, mmapFname)
		if err != nil {
			return nil, err
		}
	}

	return openMmapped(mmapFname)
}

func writeMmapped(fname string, mmapFname string) error {
	err, buf := readGDAL(fname)
	if err != nil {
		return err
	}
	if buf.xsize < 5040 || buf.xsize > 5050 || buf.ysize < 5040 || buf.ysize > 5050 {
		return fmt.Errorf("unexpected dem file buffer size %d x %d", buf.xsize, buf.ysize)
	}
	mmapdata := toMmapStruct(buf)

	var bytes = (*(*[mmapstructSize]byte)(unsafe.Pointer(mmapdata)))[:]
	return ioutil.WriteFile(mmapFname, bytes, 0644)
}

func openMmapped(fname string) (*Mmapstruct, error) {
	file, err := os.OpenFile(fname, os.O_RDONLY, 0)
	defer file.Close()

	if err != nil {
		return nil, err
	}

	data, err := syscall.Mmap(int(file.Fd()), 0, int(mmapstructSize), syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil {
		return nil, err
	}
	return (*Mmapstruct)(unsafe.Pointer(&data[0])), nil
}

