package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/tkrajina/gpxgo/gpx"
	"os"
	"strconv"
	"strings"
	"time"
)

// GeoPoint used by bryton device
type GeoPoint struct {
	lat int32
	lon int32
}

const smyInitFlag int16 = 0x01

// BrytonSmy content and layout of .smy bryton file
type BrytonSmy struct {
	smyFlag         int16
	coordinateCount int16
	bboxLatNe       int32
	bboxLatSw       int32
	bboxLonNe       int32
	bboxLonSw       int32
	totalDst        int32
}

// BrytonTrack content and layout of .track bryton file
type BrytonTrack []GeoPoint

// DirectionCodes used by Bryton device to show corresponding arrow icos
const (
	DirectionCodeCloseLeft   uint8 = 0x07
	DirectionCodeLeft        uint8 = 0x03
	DirectionCodeSlightLeft  uint8 = 0x05
	DirectionCodeGoAhead     uint8 = 0x01
	DirectionCodeSlightRight uint8 = 0x04
	DirectionCodeRight       uint8 = 0x02
	DirectionCodeCloseRight  uint8 = 0x06
)

// Waypoint represents entry used by Bryton device
type Waypoint struct {
	coordinateIndex     uint16
	directionCode       uint8
	distance            uint16
	timeSec             uint16
	waypointDescription [32]uint8
}

// BrytonTinfo content of .tinfo bryton file
type BrytonTinfo []Waypoint

// BrytonData binds all three files required for navigation using Bryton device
type BrytonData struct {
	smy   BrytonSmy
	track BrytonTrack
	tinfo BrytonTinfo
}

type Bryton interface {
	Export(outFileName string) error
}

func (s *BrytonSmy) Export(outFileName string) error {

	var binaryBuffer bytes.Buffer
	err := binary.Write(&binaryBuffer, binary.BigEndian, s)

	if err != nil {
		panic(err)
	}

	err = storeToFile(binaryBuffer, adjustFilename(outFileName, ".smy"))

	return err
}

func (t BrytonTrack) Export(outFileName string) error {

	var err error
	var binaryBuffer bytes.Buffer

	for _, trackEntry := range t {

		// 4byte - latitude
		// 4byte - longitude
		// 8byte - reserved 0x00 0x00 0x00 0x00 0x00 0x00 0x00 0x00
		layout := []interface{}{trackEntry.lat, trackEntry.lon, uint64(0x00)}

		for _, entry := range layout {
			err = binary.Write(&binaryBuffer, binary.BigEndian, entry)

			if err != nil {
				panic(err)
			}
		}

	}

	err = storeToFile(binaryBuffer, adjustFilename(outFileName, ".track"))

	return err
}

func (t BrytonTinfo) Export(outFileName string) error {

	var err error
	var binaryBuffer bytes.Buffer

	for _, tinfoEntry := range t {

		// 2byte - coordinate index
		// 1byte - direction
		// 1byte - reserved 0x00
		// 2byte - distance
		// 2byte - reserved 0x00 0x00
		// 2byte - time
		// 2byte - reserved 0x00 0x00
		// 32byte - description
		layout := []interface{}{tinfoEntry.coordinateIndex, tinfoEntry.directionCode, uint8(0x00), tinfoEntry.distance, uint16(0x00),
			tinfoEntry.timeSec, uint16(0x00), tinfoEntry.waypointDescription}

		for _, entry := range layout {
			err = binary.Write(&binaryBuffer, binary.BigEndian, entry)
			if err != nil {
				return err
			}
		}
	}

	err = storeToFile(binaryBuffer, adjustFilename(outFileName, ".tinfo"))

	return err

}

func storeToFile(buf bytes.Buffer, outFileName string) error {

	file, err := os.Create(outFileName)

	if err != nil {
		return err
	}

	defer file.Close()

	num, err := file.Write(buf.Bytes())

	if err != nil {
		fmt.Println("Failed to save " + outFileName + " error:" + err.Error())
		return err
	}

	fmt.Println(strconv.Itoa(num) + " bytes has been saved to " + outFileName)
	return err
}

func (data *BrytonData) Export(outFileName string) {

	data.smy.Export(outFileName)
	data.track.Export(outFileName)
	data.tinfo.Export(outFileName)
}

func adjustFilename(in string, extension string) string {
	out := strings.Split(in, ".")
	return out[0] + extension
}

func (d *BrytonData) ImportGpx(gpxFileName string) error {

	fmt.Println("Reading... ", gpxFileName)
	startTimestamp := time.Now()

	gpxData, err := gpx.ParseFile(gpxFileName)

	if err != nil {
		fmt.Println("Failed to parse " + gpxFileName + " error:" + err.Error())
		return err
	}

	d.smy.coordinateCount = int16(gpxData.GetTrackPointsNo())
	d.smy.totalDst = gpxData.Length3D()

	fmt.Println("...finished in ", -startTimestamp.Sub(time.Now()))
	return err
}

func main() {

	inFile := "60km_v1.gpx"

	// var data BrytonSmy
	// data.smyFlag = smyInitFlag
	// data.Export("testout")

	// dataTrack := make(BrytonTrack, 5)
	// dataTrack.Export("testout.track")

	// dataTinfo := make(BrytonTinfo, 5)
	// dataTinfo.Export("testout.track")

	var data BrytonData
	data.ImportGpx(inFile)
}
