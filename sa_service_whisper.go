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
)

type SAServiceWhisperCpp struct {
	addr string //http://127.0.0.1:8080/

	cache map[string]string //results

	last_setModel string
}

func SAServiceWhisperCpp_cachePath() string {
	return "services/whisper.cpp.json"
}

func NewSAServiceWhisperCpp(addr string) *SAServiceWhisperCpp {
	wh := &SAServiceWhisperCpp{}

	wh.addr = addr

	wh.cache = make(map[string]string)

	//load cache
	{
		js, _ := os.ReadFile(SAServiceWhisperCpp_cachePath())
		if len(js) > 0 {
			err := json.Unmarshal(js, &wh.cache)
			if err != nil {
				fmt.Printf("NewSAServiceWhisperCpp() failed: %v\n", err)
			}
		}
	}

	return wh
}
func (wh *SAServiceWhisperCpp) Destroy() {
	//save cache
	js, err := json.Marshal(wh.cache)
	if err == nil {
		os.WriteFile(SAServiceWhisperCpp_cachePath(), js, 0644)
	}
}

func (wh *SAServiceWhisperCpp) findCache(model string, blob OsBlob) (string, bool) {
	str, found := wh.cache[model+blob.hash.Hex()]
	return str, found
}
func (wh *SAServiceWhisperCpp) addCache(model string, blob OsBlob, value string) {
	wh.cache[model+blob.hash.Hex()] = value
}

func (wh *SAServiceWhisperCpp) Translate(model string, blob OsBlob) (string, float64, bool, error) {
	//find blob
	str, found := wh.findCache(model, blob)
	if found {
		return str, 1, true, nil
	}

	//set model
	if model != wh.last_setModel {
		err := wh.setModel(model)
		if err != nil {
			return "", 0, false, fmt.Errorf("setModel() failed: %w", err)
		}
	}

	//translate
	str, err := wh.translate(blob)
	if err != nil {
		return "", 0, false, fmt.Errorf("translate() failed: %w", err)
	}

	wh.addCache(model, blob, str)
	return str, 1, true, nil
}

func (wh *SAServiceWhisperCpp) setModel(model string) error {

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

func (wh *SAServiceWhisperCpp) translate(blob OsBlob) (string, error) {

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	//set parameters
	{
		part, err := writer.CreateFormFile("file", "audio.wav")
		if err != nil {
			return "", fmt.Errorf("CreateFormFile() failed: %w", err)
		}
		part.Write(blob.data)
		//writer.WriteField("temperature", "0.0")
		//writer.WriteField("temperature_inc", "0.2")
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
