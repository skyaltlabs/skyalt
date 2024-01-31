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
	"mime/multipart"
	"net/http"
	"os"
	"sync"
	"time"
)

type SAService_WhisperQue struct {
	model string
	blob  OsBlob
}

type SAService_Whisper struct {
	addr string //http://127.0.0.1:8080/

	mu    sync.Mutex
	que   []SAService_WhisperQue
	cache map[string]string //results

	last_setModel string
}

func SAService_Whisper_cachePath() string {
	return "services/whisper/cache.json"
}

func NewSAService_Whisper(addr string) *SAService_Whisper {
	wh := &SAService_Whisper{}

	wh.addr = addr

	wh.cache = make(map[string]string)

	//load cache
	{
		js, _ := os.ReadFile(SAService_Whisper_cachePath())
		if len(js) > 0 {
			err := json.Unmarshal(js, &wh.cache)
			if err != nil {
				fmt.Printf("NewSAService_Whisper() failed: %v\n", err)
			}
		}
	}

	go wh.tick() //run service in 2nd thread

	return wh
}
func (wh *SAService_Whisper) Destroy() {

	//wait for tick() thread to finish? ...

	js, err := json.Marshal(wh.cache)
	if err == nil {
		os.WriteFile(SAService_Whisper_cachePath(), js, 0644)
	}
}

func (wh *SAService_Whisper) findCache(model string, blob OsBlob) (string, bool) {
	wh.mu.Lock()
	defer wh.mu.Unlock()

	str, found := wh.cache[model+blob.hash.Hex()]
	return str, found
}
func (wh *SAService_Whisper) addCache(model string, blob OsBlob, value string) {
	wh.mu.Lock()
	defer wh.mu.Unlock()

	wh.cache[model+blob.hash.Hex()] = value
}

func (wh *SAService_Whisper) addQue(model string, blob OsBlob) {
	wh.mu.Lock()
	defer wh.mu.Unlock()

	//find
	for _, q := range wh.que {
		if q.model == model && q.blob.CmpHash(&blob) {
			return //already exist
		}
	}

	//find
	wh.que = append(wh.que, SAService_WhisperQue{model: model, blob: blob})
}
func (wh *SAService_Whisper) getFirstQue() (SAService_WhisperQue, bool) {
	wh.mu.Lock()
	defer wh.mu.Unlock()

	if len(wh.que) > 0 {
		return wh.que[0], true //found
	}
	return SAService_WhisperQue{}, false //not found
}
func (wh *SAService_Whisper) removeFirstQue() {
	wh.mu.Lock()
	defer wh.mu.Unlock()

	if len(wh.que) > 0 {
		wh.que = wh.que[1:]
	}
}

func (wh *SAService_Whisper) Translate(model string, blob OsBlob) (string, float64, bool, error) {
	//find blob
	str, found := wh.findCache(model, blob)
	if found {
		return str, 1, true, nil
	}

	wh.addQue(model, blob)
	return "", 0.5, false, nil
}

func (wh *SAService_Whisper) tick() {
	for {
		que, ok := wh.getFirstQue()
		if ok {
			//set model
			if que.model != wh.last_setModel {
				err := wh.setModel(que.model)
				if err != nil {
					fmt.Println("setModel() error:", err.Error())
					wh.removeFirstQue()
					continue
				}
			}

			//translace
			str, err := wh.translate(que.blob)
			if err != nil {
				fmt.Println("setModel() error:", err.Error())
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

func (wh *SAService_Whisper) setModel(model string) error {

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	writer.WriteField("model", model)
	writer.Close()

	req, err := http.NewRequest(http.MethodPost, wh.addr+"load", body)
	if err != nil {
		return fmt.Errorf("NewRequest() failed: %w", err)
	}
	req.Header.Add("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("Do() failed: %w", err)
	}

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("ReadAll() failed: %w", err)
	}

	if res.StatusCode != 200 {
		return fmt.Errorf("statusCode != 200, response: %s", resBody)
	}

	wh.last_setModel = model
	return nil
}

func (wh *SAService_Whisper) translate(blob OsBlob) (string, error) {

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	//set parameters
	{
		part, err := writer.CreateFormFile("file", "audio.wav")
		if err != nil {
			return "", fmt.Errorf("CreateFormFile() failed: %w", err)
		}
		part.Write(blob.data)
		writer.WriteField("temperature", "0.0")
		writer.WriteField("temperature_inc", "0.2")
		writer.WriteField("response_format", "verbose_json")
	}
	writer.Close()

	req, err := http.NewRequest(http.MethodPost, wh.addr+"inference", body)
	if err != nil {
		return "", fmt.Errorf("NewRequest() failed: %w", err)
	}
	req.Header.Add("Content-Type", writer.FormDataContentType())

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
