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
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

func SAExe_Code_python(node *SANode) bool {

	codeAttr := node.GetAttr("code", "")

	code := codeAttr.GetString()
	if code == "" {
		codeAttr.SetErrorExe("empty")
		return false
	}

	if strings.Contains(strings.ToLower(code), "import") {
		codeAttr.SetErrorExe("Code contains 'import' keyword")
		return false
	}

	attrsList := make(map[string]interface{})
	for _, a := range node.Attrs {
		if a.Name == "code" {
			continue //skip
		}
		attrsList[a.Name] = a.GetResult().value
	}

	type Pyth struct {
		Code  string                 `json:"code"`
		Attrs map[string]interface{} `json:"attrs"`
	}
	jsonBody, err := json.Marshal(Pyth{Code: code, Attrs: attrsList})
	if err != nil {
		node.SetError("Marshal() failed: " + err.Error())
		return false
	}
	//jsonBody := `{"code": "out = x + y", "attrs": {"x":4, "y":6}}`

	body := bytes.NewReader([]byte(jsonBody))
	req, err := http.NewRequest(http.MethodPost, "http://localhost:8092", body)
	if err != nil {
		node.SetError("NewRequest() failed: " + err.Error())
		return false
	}
	req.Header.Add("Content-Type", "application/json")

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return false
	}

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		node.SetError("ReadAll() failed: " + err.Error())
		return false
	}

	if res.StatusCode != 200 {
		node.SetError(fmt.Sprintf("Server return StatusCode: %d", res.StatusCode))
		return false
	}

	outputs := make(map[string]interface{})
	err = json.Unmarshal(resBody, &outputs)
	if err != nil {
		node.SetError("Unmarshal() failed: " + err.Error())
		return false
	}

	for name, value := range outputs {
		if strings.EqualFold(name, "code") {
			continue //skip
		}
		attr := node.GetAttr(name, "")
		switch vv := value.(type) {
		case string:
			attr.SetExpString(vv, false)
		case float64:
			attr.SetExpFloat(vv)
		case int:
			attr.SetExpInt(vv)
		default:
			fmt.Println("Unsupported format")
		}
	}

	return true
}
