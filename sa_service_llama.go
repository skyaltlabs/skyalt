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
	"sync"
)

type SAServiceLLamaCppProps struct {
	Prompt            string   `json:"prompt"`
	Seed              int      `json:"seed"`
	N_predict         int      `json:"n_predict"`
	Temperature       float64  `json:"temperature"`
	Dynatemp_range    float64  `json:"dynatemp_range"`
	Dynatemp_exponent float64  `json:"dynatemp_exponent"`
	Stop              []string `json:"stop"`
	Repeat_last_n     int      `json:"repeat_last_n"`
	Repeat_penalty    float64  `json:"repeat_penalty"`
	Top_k             int      `json:"top_k"`
	Top_p             float64  `json:"top_p"`
	Min_p             float64  `json:"min_p"`
	Tfs_z             float64  `json:"tfs_z"`
	Typical_p         float64  `json:"typical_p"`
	Presence_penalty  float64  `json:"presence_penalty"`
	Frequency_penalty float64  `json:"frequency_penalty"`
	Mirostat          int      `json:"mirostat"`
	Mirostat_tau      float64  `json:"mirostat_tau"`
	Mirostat_eta      float64  `json:"mirostat_eta"`
	//Grammar           string   `json:"grammar"` //[]string?
	N_probs int `json:"n_probs"`
	//Image_data //{"data": "<BASE64_STRING>", "id": 12}
	Cache_prompt bool `json:"cache_prompt"`
	Slot_id      int  `json:"slot_id"`

	//Stream bool ......
}

func (p *SAServiceLLamaCppProps) Hash() (OsHash, error) {
	js, err := json.Marshal(p)
	if err != nil {
		return OsHash{}, err
	}
	return InitOsHash(js)
}

type SAServiceLLamaCpp struct {
	addr string //http://127.0.0.1:8080/

	cache      map[string]string //results
	cache_lock sync.Mutex        //for cache
}

func SAServiceLLamaCpp_cachePath() string {
	return "services/llama.cpp.json"
}

func NewSAServiceLLamaCpp(addr string, port string) *SAServiceLLamaCpp {
	wh := &SAServiceLLamaCpp{}

	wh.addr = addr + ":" + port

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

func (wh *SAServiceLLamaCpp) FindCache(model string, propsHash OsHash) (string, bool) {
	wh.cache_lock.Lock()
	defer wh.cache_lock.Unlock()

	str, found := wh.cache[model+propsHash.Hex()]
	return str, found
}
func (wh *SAServiceLLamaCpp) addCache(model string, propsHash OsHash, value string) {
	wh.cache_lock.Lock()
	defer wh.cache_lock.Unlock()

	wh.cache[model+propsHash.Hex()] = value
}

func (wh *SAServiceLLamaCpp) Complete(model string, props *SAServiceLLamaCppProps) (string, float64, bool, error) {
	//find
	propsHash, err := props.Hash()
	if err != nil {
		return "", 0, false, fmt.Errorf("Hash() failed: %w", err)
	}
	str, found := wh.FindCache(model, propsHash)
	if found {
		return str, 1, true, nil
	}

	str, err = wh.complete(props)
	if err != nil {
		return "", 0, false, fmt.Errorf("complete() failed: %w", err)
	}

	wh.addCache(model, propsHash, str)
	return str, 1, true, nil
}

func (wh *SAServiceLLamaCpp) complete(props *SAServiceLLamaCppProps) (string, error) {
	js, err := json.Marshal(props)
	if err != nil {
		return "", fmt.Errorf("Marshal() failed: %w", err)
	}

	body := bytes.NewReader([]byte(js))

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
