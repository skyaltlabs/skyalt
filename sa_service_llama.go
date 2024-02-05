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
	"os"
)

type SAServiceLLamaCpp struct {
	addr string //http://127.0.0.1:8080/

	cache map[string]string //results
}

func SAServiceLLamaCpp_cachePath() string {
	return "services/llama.cpp.json"
}

func NewSAServiceLLamaCpp(addr string) *SAServiceLLamaCpp {
	wh := &SAServiceLLamaCpp{}

	wh.addr = addr

	wh.cache = make(map[string]string)

	//load cache
	{
		js, _ := os.ReadFile(SAServiceLLamaCpp_cachePath())
		if len(js) > 0 {
			err := json.Unmarshal(js, &wh.cache)
			if err != nil {
				fmt.Printf("NewSAServiceLLamaCpp() failed: %v\n", err)
			}
		}
	}

	return wh
}
func (wh *SAServiceLLamaCpp) Destroy() {
	//save cache
	js, err := json.Marshal(wh.cache)
	if err == nil {
		os.WriteFile(SAServiceLLamaCpp_cachePath(), js, 0644)
	}
}

func (wh *SAServiceLLamaCpp) findCache(model string, blob OsBlob) (string, bool) {
	str, found := wh.cache[model+blob.hash.Hex()]
	return str, found
}
func (wh *SAServiceLLamaCpp) addCache(model string, blob OsBlob, value string) {
	wh.cache[model+blob.hash.Hex()] = value
}

func (wh *SAServiceLLamaCpp) Complete(model string, blob OsBlob) (string, float64, bool, error) {
	//find blob
	str, found := wh.findCache(model, blob)
	if found {
		return str, 1, true, nil
	}

	str, err := wh.complete(blob)
	if err != nil {
		return "", 0, false, fmt.Errorf("complete() failed: %w", err)
	}

	wh.addCache(model, blob, str)
	return str, 1, true, nil
}

func (wh *SAServiceLLamaCpp) complete(blob OsBlob) (string, error) {

	aa := OsText_PrintToRaw(string(blob.data))

	//stream = true ............
	jsonBody := fmt.Sprintf(`{
		"stream": false,
		"n_predict": 400,
		"temperature": 0.7,
		"stop": ["</s>", "Llama:", "User:"],
		"repeat_last_n": 256,
		"repeat_penalty": 1.18,
		"top_k": 40,
		"top_p": 0.5,
		"min_p": 0.05,
		"tfs_z": 1,
		"typical_p": 1,
		"presence_penalty": 0,
		"frequency_penalty": 0,
		"mirostat": 0,
		"mirostat_tau": 5,
		"mirostat_eta": 0.1,
		"grammar": "",
		"n_probs": 0,
		"image_data": [],
		"cache_prompt": true,
		"slot_id": 0,
		"prompt": "%s"
		}`, aa)

	body := bytes.NewReader([]byte(jsonBody))

	req, err := http.NewRequest(http.MethodPost, wh.addr+"completion", body)
	if err != nil {
		return "", fmt.Errorf("NewRequest() failed: %w", err)
	}
	req.Header.Add("Content-Type", "application/json")

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("Do() failed: %w", err)
	}

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return "", fmt.Errorf("ReadAll() failed: %w", err)
	}

	if res.StatusCode != 200 {
		return "", fmt.Errorf("statusCode != 200, response: %s", resBody)
	}

	return string(resBody), nil
}
