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
)

type SAServicePython struct {
	addr string //http://127.0.0.1:8080/
}

func NewSAServicePython(addr string) *SAServicePython {
	py := &SAServicePython{}

	py.addr = addr

	return py
}
func (py *SAServicePython) Destroy() {
}

func (py *SAServicePython) Exec(code string, attrsList map[string]interface{}) (map[string]interface{}, string, error) {

	type Pyth struct {
		Code  string                 `json:"code"`
		Attrs map[string]interface{} `json:"attrs"`
	}
	jsonBody, err := json.Marshal(Pyth{Code: code, Attrs: attrsList})
	if err != nil {
		return nil, "", fmt.Errorf("Marshal() failed: %v", err)
	}

	// send code and attributes
	body := bytes.NewReader([]byte(jsonBody))
	req, err := http.NewRequest(http.MethodPost, py.addr, body)
	if err != nil {
		return nil, "", fmt.Errorf("NewRequest() failed: %v", err)
	}
	req.Header.Add("Content-Type", "application/json")
	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("Do() failed: %v", err)
	}

	// recv json
	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, "", fmt.Errorf("ReadAll() failed: %v", err)
	}

	if res.StatusCode != 200 {
		return nil, "", fmt.Errorf("server return StatusCode: %d", res.StatusCode)

	}

	// unpacked json
	type Ret struct {
		Attrs map[string]interface{}
		Err   string
	}
	var out Ret
	out.Attrs = make(map[string]interface{})
	err = json.Unmarshal(resBody, &out)
	if err != nil {
		return nil, "", fmt.Errorf("Unmarshal() failed: %v", err)
	}

	return out.Attrs, out.Err, nil //ok
}
