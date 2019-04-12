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

func (t BrytonTrack) getCoordinateIndex(point GeoPoint) uint16 {
	for i, entry := range t {
		if entry == point {
			return uint16(i)
		}
	}

	return uint16(0)
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

	// smy data
	d.smy.coordinateCount = int16(gpxData.GetTrackPointsNo())
	d.smy.totalDst = int32(gpxData.Length3D() * 1000.0)

	d.smy.bboxLatNe = adjustGeoCoordinates(gpxData.Bounds().MaxLatitude)
	d.smy.bboxLonNe = adjustGeoCoordinates(gpxData.Bounds().MaxLongitude)
	d.smy.bboxLatSw = adjustGeoCoordinates(gpxData.Bounds().MinLatitude)
	d.smy.bboxLonSw = adjustGeoCoordinates(gpxData.Bounds().MinLongitude)

	// track data
	if len(gpxData.Tracks) > 0 {
		if len(gpxData.Tracks[0].Segments) > 0 {

			for _, p := range gpxData.Tracks[0].Segments[0].Points {
				d.track = append(d.track, GeoPoint{adjustGeoCoordinates(p.Point.GetLatitude()), adjustGeoCoordinates(p.Point.GetLongitude())})
			}
		}
	}

	// tinfo data
	for _, w := range gpxData.Waypoints {
		var wpt Waypoint
		wpt.coordinateIndex = d.track.getCoordinateIndex(GeoPoint{adjustGeoCoordinates(w.Point.GetLatitude()), adjustGeoCoordinates(w.Point.GetLongitude())})
		wpt.directionCode = convertDirectionCode(strings.ToLower(w.Symbol))
		// d.tinfo w.Type
		d.tinfo = append(d.tinfo, wpt)
	}
	// gpxData.Waypoints[0]
	// var wpt Waypoint
	// wpt.coordinateIndex
	// wpt.directionCode
	// wpt.distance
	// wpt.timeSec
	// wpt.waypointDescription

	// d.tinfo

	fmt.Println("...finished in ", -startTimestamp.Sub(time.Now()))
	return err
}

func adjustGeoCoordinates(geo float64) int32 {
	return int32(geo * 1000000.0)
}

func convertDirectionCode(gpxDirCode string) uint8 {

	brytonDirCode := DirectionCodeGoAhead

	switch gpxDirCode {
	case "tshl":
		brytonDirCode = DirectionCodeCloseLeft
	case "left":
		brytonDirCode = DirectionCodeLeft
	case "tsll":
		brytonDirCode = DirectionCodeSlightLeft
	case "straight":
		brytonDirCode = DirectionCodeGoAhead
	case "tslr":
		brytonDirCode = DirectionCodeSlightRight
	case "right":
		brytonDirCode = DirectionCodeRight
	case "tshr":
		brytonDirCode = DirectionCodeCloseRight
	default:
		fmt.Println("Unsupported direction code: " + gpxDirCode + "! Using GoAhead!")
	}

	return brytonDirCode
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
