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
	"bytes"
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"strings"
)

type SAExe_ConvertTrkseg struct {
	Trkpt []struct {
		Lat  float32 `xml:"lat,attr" json:"lat"`
		Lon  float32 `xml:"lon,attr" json:"lon"`
		Ele  float32 `xml:"ele" json:"ele,omitempty"`
		Time string  `xml:"time" json:"time,omitempty"`
	} `xml:"trkpt"`
}

type SAExe_ConvertGPX struct {
	Trkseg []SAExe_ConvertTrkseg `xml:"trk>trkseg" json:"segments"`
}

func SAExe_Convert_GpxToJson(node *SANode) bool {

	gpxAttr := node.GetAttr("gpx", "")
	jsonAttr := node.GetAttr("_json", "")

	gpx := gpxAttr.GetBlob()
	if gpx.Len() == 0 {
		return true //empty in, empty out
	}

	//gpx -> struct
	var g SAExe_ConvertGPX
	err := xml.Unmarshal(gpx.data, &g)
	if err != nil {
		gpxAttr.SetError(fmt.Errorf("Unmarshal() failed: %w", err))
		return false
	}

	//struct -> json
	js, err := json.Marshal(g.Trkseg)
	if err != nil {
		node.SetError(fmt.Errorf("Marshal() failed: %w", err))
		return false
	}

	jsonAttr.SetOutBlob(js)
	return true
}

func SAExe_Convert_CsvToJson(node *SANode) bool {

	csvAttr := node.GetAttr("Csv", "")
	firstLineHeader := node.GetAttrUi("first_line_header", 1, SAAttrUi_SWITCH).GetBool()
	resultAttr := node.GetAttr("_result", []byte("[]"))

	csvBlob := csvAttr.GetBlob()
	if csvBlob.Len() == 0 {
		return true //empty in, empty out
	}

	data, err := csv.NewReader(bytes.NewBuffer(csvBlob.data)).ReadAll()
	if err != nil {
		node.SetError(fmt.Errorf("ReadAll() failed: %w", err))
	}

	max_cols := 0
	for _, ln := range data {
		max_cols = OsMax(max_cols, len(ln))
	}

	rws := "["

	if max_cols > 0 {
		//create columns list
		var columnNames []string
		if firstLineHeader {
			columnNames = append(columnNames, data[0]...)
		}
		for i := len(columnNames); i < max_cols; i++ {
			columnNames = append(columnNames, fmt.Sprintf("c%d", i))
		}

		//lines
		for i, ln := range data {
			if firstLineHeader && i == 0 {
				continue //skip header
			}

			//items
			rws += "["
			for _, str := range ln {
				rws += "\"" + str + "\","
			}
			rws, _ = strings.CutSuffix(rws, ",")
			rws += "],"
		}
	}

	rws, _ = strings.CutSuffix(rws, ",")
	rws += "]"

	resultAttr.SetOutBlob([]byte(rws))
	return true
}
