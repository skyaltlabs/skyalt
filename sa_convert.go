/*
Copyright 2023 Milan Suk

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this db except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
)

type SAConvertGPX struct {
	Trkseg []SAConvertTrkseg `xml:"trk>trkseg" json:"segments"`
}

type SAConvertTrkseg struct {
	Trkpt []struct {
		Lat  float32 `xml:"lat,attr" json:"lat"`
		Lon  float32 `xml:"lon,attr" json:"lon"`
		Ele  float32 `xml:"ele" json:"ele,omitempty"`
		Time string  `xml:"time" json:"time,omitempty"`
	} `xml:"trkpt"`
}

func (node *SANode) SAConvert_GpxToJson() bool {

	gpxAttr := node.GetAttr("gpx", "")
	jsonAttr := node.GetAttrOutput("json", "")
	jsonAttr.result.SetBlob(nil) //reset

	gpx := gpxAttr.result.Blob()
	if len(gpx) == 0 {
		return true //empty in, empty out
	}

	//gpx -> struct
	var g SAConvertGPX
	err := xml.Unmarshal(gpx, &g)
	if err != nil {
		gpxAttr.SetErrorExe(fmt.Sprintf("Unmarshal() failed: %v", err))
		return false
	}

	//struct -> json
	js, err := json.Marshal(g.Trkseg)
	if err != nil {
		node.SetError(fmt.Sprintf("Marshal() failed: %v", err))
		return false
	}

	jsonAttr.result.SetBlob(js)
	return true
}
