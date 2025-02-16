package brytongo

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

// Default value of first byte in .smy file
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

// Export BrytonSmy structure to .smy file
func (s *BrytonSmy) Export(outFileName string) error {

	var err error
	var binaryBuffer bytes.Buffer

	layout := []interface{}{int16(0x01), s.coordinateCount, s.bboxLatNe, s.bboxLatSw, s.bboxLonNe, s.bboxLonSw, s.totalDst}

	for _, entry := range layout {
		err = binary.Write(&binaryBuffer, binary.LittleEndian, entry)
		if err != nil {
			panic(err)
		}
	}

	err = storeToFile(binaryBuffer, adjustFilename(outFileName, ".smy"))

	return err
}

// Export BrytonTrack structure to .track file
func (t BrytonTrack) Export(outFileName string) error {

	var err error
	var binaryBuffer bytes.Buffer

	for _, trackEntry := range t {

		// 4byte - latitude
		// 4byte - longitude
		// 8byte - reserved 0x00 0x00 0x00 0x00 0x00 0x00 0x00 0x00
		layout := []interface{}{trackEntry.lat, trackEntry.lon, uint64(0x00)}

		for _, entry := range layout {
			err = binary.Write(&binaryBuffer, binary.LittleEndian, entry)

			if err != nil {
				panic(err)
			}
		}

	}

	err = storeToFile(binaryBuffer, adjustFilename(outFileName, ".track"))

	return err
}

// Returns track index of searched geo point.
// If not found, returns 0
func (t BrytonTrack) getCoordinateIndex(point GeoPoint) uint16 {
	for i, entry := range t {
		if entry == point {
			return uint16(i)
		}
	}

	return uint16(0)
}

// Export BrytonTinfo structure to .tinfo file
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
			err = binary.Write(&binaryBuffer, binary.LittleEndian, entry)
			if err != nil {
				return err
			}
		}
	}

	err = storeToFile(binaryBuffer, adjustFilename(outFileName, ".tinfo"))

	return err

}

// Stores byte buffer to file
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

// Export BrytonData structure to .smy .track and .tinfo files
func (d *BrytonData) Export(outFileName string) {

	d.smy.Export(outFileName)
	d.track.Export(outFileName)
	d.tinfo.Export(outFileName)
}

// Strips extension from in filename and adds passed as argument
func adjustFilename(in string, extension string) string {
	out := strings.Split(in, ".")
	return out[0] + extension
}

// ImportGpx file and parse to BrytonData structure
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
	fmt.Printf("Coordinate count: %v\n", d.smy.coordinateCount)

	d.smy.totalDst = int32(gpxData.Length3D())
	fmt.Printf("Total distance %0.2fkm\n", gpxData.Length3D()/1000.0)

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
	fmt.Printf("Waypoint count: %v\n", len(gpxData.Waypoints))

	for _, w := range gpxData.Waypoints {
		var wpt Waypoint
		wpt.coordinateIndex = d.track.getCoordinateIndex(GeoPoint{adjustGeoCoordinates(w.Point.GetLatitude()), adjustGeoCoordinates(w.Point.GetLongitude())})
		wpt.directionCode = convertDirectionCode(strings.ToLower(w.Symbol))

		// TODO: should we use these fields?
		wpt.distance = 0
		wpt.timeSec = 0

		copy(wpt.waypointDescription[:], w.Name)

		d.tinfo = append(d.tinfo, wpt)
	}

	fmt.Println("...finished in ", -startTimestamp.Sub(time.Now()))
	return err
}

// adjustGeoCoordinates from float to Bryton compliant
func adjustGeoCoordinates(geo float64) int32 {
	return int32(geo * 1000000.0)
}

// Convert gpx waypoint direction markers to Bryton compliant.
// Currently only GPSies.com markers are supported
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
