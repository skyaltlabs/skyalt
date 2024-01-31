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
	"time"
)

type SAServiceLLamaCppQue struct {
	model string
	blob  OsBlob
}

type SAServiceLLamaCpp struct {
	addr string //http://127.0.0.1:8080/

	mu    sync.Mutex
	que   []SAServiceLLamaCppQue
	cache map[string]string //results
}

func SAServiceLLamaCpp_cachePath() string {
	return "services/llama/cache.json"
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

	go wh.tick() //run service in 2nd thread

	return wh
}
func (wh *SAServiceLLamaCpp) Destroy() {

	//wait for tick() thread to finish? ...

	js, err := json.Marshal(wh.cache)
	if err == nil {
		os.WriteFile(SAServiceLLamaCpp_cachePath(), js, 0644)
	}
}

func (wh *SAServiceLLamaCpp) findCache(model string, blob OsBlob) (string, bool) {
	wh.mu.Lock()
	defer wh.mu.Unlock()

	str, found := wh.cache[model+blob.hash.Hex()]
	return str, found
}
func (wh *SAServiceLLamaCpp) addCache(model string, blob OsBlob, value string) {
	wh.mu.Lock()
	defer wh.mu.Unlock()

	wh.cache[model+blob.hash.Hex()] = value
}

func (wh *SAServiceLLamaCpp) addQue(model string, blob OsBlob) {
	wh.mu.Lock()
	defer wh.mu.Unlock()

	//find
	for _, q := range wh.que {
		if q.model == model && q.blob.CmpHash(&blob) {
			return //already exist
		}
	}

	//find
	wh.que = append(wh.que, SAServiceLLamaCppQue{model: model, blob: blob})
}
func (wh *SAServiceLLamaCpp) getFirstQue() (SAServiceLLamaCppQue, bool) {
	wh.mu.Lock()
	defer wh.mu.Unlock()

	if len(wh.que) > 0 {
		return wh.que[0], true //found
	}
	return SAServiceLLamaCppQue{}, false //not found
}
func (wh *SAServiceLLamaCpp) removeFirstQue() {
	wh.mu.Lock()
	defer wh.mu.Unlock()

	if len(wh.que) > 0 {
		wh.que = wh.que[1:]
	}
}

func (wh *SAServiceLLamaCpp) Complete(model string, blob OsBlob) (string, float64, bool, error) {
	//find blob
	str, found := wh.findCache(model, blob)
	if found {
		return str, 1, true, nil
	}

	wh.addQue(model, blob)
	return "", 0.5, false, nil
}

func (wh *SAServiceLLamaCpp) tick() {
	for {
		que, ok := wh.getFirstQue()
		if ok {
			//translace
			str, err := wh.complete(que.blob)
			if err != nil {
				fmt.Println("complete() error:", err.Error())
				wh.removeFirstQue()
				continue
			}

			wh.addCache(que.model, que.blob, str)
			wh.removeFirstQue()
		} else {
			time.Sleep(time.Millisecond * 10)
		}
	}
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
