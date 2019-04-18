# brytongo

Simple library allowing track-and-waypoints export from gpx to bryton binary (.smy .track .tinfo) files.
Currently supports tracks created in GPSies.com only. Tested with Bryton Rider 330.

## Example usage:
```go
package main

import (
	"flag"
	"github.com/kapzzzz/brytongo"
)

func main() {

	inFileNamePtr := flag.String("i", "", "input file name")
	outFileNamePtr := flag.String("o", "out", "output file name")
	flag.Parse()

	var data brytongo.BrytonData
	err := data.ImportGpx(*inFileNamePtr)
	if err == nil {
		data.Export(*outFileNamePtr)
	}

}
```
