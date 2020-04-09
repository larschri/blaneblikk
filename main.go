package main

import (
	"fmt"
)

func main() {
	err, buf := ReadGDAL("dem-files/6603_1_10m_z32.dem")
	if err != nil {
		panic(err)
	}
	fmt.Println(buf.buffer[0:10])
}
