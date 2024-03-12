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
	Model    string         `json:"model"`
	Messages []SAServiceMsg `json:"messages"`

	//Prompt            string   `json:"prompt"`
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
	Mirostat          bool     `json:"mirostat"` //not int?
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
	services *SAServices
	addr     string //http://127.0.0.1:8080/

	cache      map[string][]byte //results
	cache_lock sync.Mutex        //for cache
}

func SAServiceLLamaCpp_cachePath() string {
	return "services/llama.cpp.json"
}

func NewSAServiceLLamaCpp(services *SAServices, addr string, port string) *SAServiceLLamaCpp {
	wh := &SAServiceLLamaCpp{services: services}

	wh.addr = addr + ":" + port + "/"

	wh.cache = make(map[string][]byte)

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

func (wh *SAServiceLLamaCpp) FindCache(propsHash OsHash) ([]byte, bool) {
	wh.cache_lock.Lock()
	defer wh.cache_lock.Unlock()

	str, found := wh.cache[propsHash.Hex()]
	return str, found
}
func (wh *SAServiceLLamaCpp) addCache(propsHash OsHash, value []byte) {
	wh.cache_lock.Lock()
	defer wh.cache_lock.Unlock()

	wh.cache[propsHash.Hex()] = value
}

func (wh *SAServiceLLamaCpp) Complete(props *SAServiceLLamaCppProps) ([]byte, error) {
	//find
	propsHash, err := props.Hash()
	if err != nil {
		return nil, fmt.Errorf("Hash() failed: %w", err)
	}
	str, found := wh.FindCache(propsHash)
	if found {
		return str, nil
	}

	out, err := wh.complete(props)
	if err != nil {
		return nil, fmt.Errorf("complete() failed: %w", err)
	}

	wh.addCache(propsHash, out)
	return out, nil
}

func (wh *SAServiceLLamaCpp) complete(props *SAServiceLLamaCppProps) ([]byte, error) {
	js, err := json.Marshal(props)
	if err != nil {
		return nil, fmt.Errorf("Marshal() failed: %w", err)
	}

	body := bytes.NewReader([]byte(js))

	req, err := http.NewRequest(http.MethodPost, wh.addr+"v1/chat/completions", body)
	if err != nil {
		return nil, fmt.Errorf("NewRequest() failed: %w", err)
	}
	req.Header.Add("Content-Type", "application/json")

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Do() failed: %w", err)
	}

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("ReadAll() failed: %w", err)
	}

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("statusCode != 200, response: %s", resBody)
	}

	type STMsg struct {
		Content string
	}
	type STChoice struct {
		Message STMsg
	}
	type ST struct {
		Choices []STChoice
	}
	var answer ST
	err = json.Unmarshal(resBody, &answer)
	if err != nil {
		return nil, fmt.Errorf("Unmarshal() failed: %w", err)
	}

	if len(answer.Choices) == 0 {
		return nil, fmt.Errorf("answer is empty")
	}

	return []byte(answer.Choices[0].Message.Content), nil
}
