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
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"time"
)

type SAServiceWhisperCppProps struct {
	Offset_t    int
	Offset_n    int
	Duration    int
	Max_context int
	Max_len     int
	Best_of     int
	Beam_size   int

	Word_thold    float64
	Entropy_thold float64
	Logprob_thold float64

	Translate     bool
	Diarize       bool
	Tinydiarize   bool
	Split_on_word bool
	No_timestamps bool

	Language        string
	Detect_language bool

	Temperature     float64
	Temperature_inc float64

	Response_format string
}

func (p *SAServiceWhisperCppProps) Hash() (OsHash, error) {
	js, err := json.Marshal(p)
	if err != nil {
		return OsHash{}, err
	}
	return InitOsHash(js)
}
func (p *SAServiceWhisperCppProps) Write(w *multipart.Writer) {
	w.WriteField("offset_t", strconv.Itoa(p.Offset_t))
	w.WriteField("offset_n", strconv.Itoa(p.Offset_n))
	w.WriteField("duration", strconv.Itoa(p.Duration))
	w.WriteField("max_context", strconv.Itoa(p.Max_context))
	w.WriteField("max_len", strconv.Itoa(p.Max_len))
	w.WriteField("best_of", strconv.Itoa(p.Best_of))
	w.WriteField("beam_size", strconv.Itoa(p.Beam_size))

	w.WriteField("word_thold", strconv.FormatFloat(p.Word_thold, 'f', -1, 64))
	w.WriteField("entropy_thold", strconv.FormatFloat(p.Entropy_thold, 'f', -1, 64))
	w.WriteField("logprob_thold", strconv.FormatFloat(p.Logprob_thold, 'f', -1, 64))

	w.WriteField("translate", OsTrnString(p.Translate, "1", "0"))
	w.WriteField("diarize", OsTrnString(p.Diarize, "1", "0"))
	w.WriteField("tinydiarize", OsTrnString(p.Tinydiarize, "1", "0"))
	w.WriteField("split_on_word", OsTrnString(p.Split_on_word, "1", "0"))
	w.WriteField("no_timestamps", OsTrnString(p.No_timestamps, "1", "0"))

	w.WriteField("language", p.Language)
	w.WriteField("detect_language", OsTrnString(p.Detect_language, "1", "0"))

	w.WriteField("temperature", strconv.FormatFloat(p.Temperature, 'f', -1, 64))
	w.WriteField("temperature_inc", strconv.FormatFloat(p.Temperature_inc, 'f', -1, 64))

	w.WriteField("response_format", p.Response_format)
}

type SAServiceWhisperCpp struct {
	services *SAServices
	cmd      *exec.Cmd
	addr     string //http://127.0.0.1:8080/

	cache      map[string][]byte //results
	cache_lock sync.Mutex        //for cache

	last_setModel string
}

func SAServiceWhisperCpp_cachePath() string {
	return "services/whisper.cpp.json"
}

func NewSAServiceWhisperCpp(services *SAServices, addr string, port string, init_model string) (*SAServiceWhisperCpp, error) {
	wh := &SAServiceWhisperCpp{services: services}

	wh.addr = addr + ":" + port + "/"

	wh.cache = make(map[string][]byte)

	//load cache
	{
		js, _ := os.ReadFile(SAServiceWhisperCpp_cachePath())
		if len(js) > 0 {
			err := json.Unmarshal(js, &wh.cache)
			if err != nil {
				return nil, fmt.Errorf("NewSAServiceWhisperCpp() failed: %w", err)
			}
		}
	}

	//run process
	modelPath := "models/" + init_model + ".bin"
	{
		wh.cmd = exec.Command("./server", "--port", port, "--convert", "-m", modelPath)
		wh.cmd.Dir = "services/whisper.cpp/"

		wh.cmd.Stdout = os.Stdout
		wh.cmd.Stderr = os.Stderr
		err := wh.cmd.Start()
		if err != nil {
			return nil, fmt.Errorf("Command() failed: %w", err)
		}
	}

	//wait until it's running
	{
		err := errors.New("err")
		st := OsTicks()
		for err != nil && OsIsTicksIn(st, 10000) { //max 10sec to start
			err = wh.setModel(modelPath)
			time.Sleep(200 * time.Millisecond)
		}
		if err != nil {
			return nil, err
		}
		wh.last_setModel = init_model
	}

	return wh, nil
}
func (wh *SAServiceWhisperCpp) Destroy() {
	//save cache
	js, err := json.Marshal(wh.cache)
	if err == nil {
		os.WriteFile(SAServiceWhisperCpp_cachePath(), js, 0644)
	}

	//py.cmd.Process.Signal(syscall.SIGQUIT)
	err = wh.cmd.Process.Kill()
	if err != nil {
		fmt.Println(err)
	}
}

func (wh *SAServiceWhisperCpp) FindCache(model string, blob OsBlob, propsHash OsHash) ([]byte, bool) {
	wh.cache_lock.Lock()
	defer wh.cache_lock.Unlock()

	str, found := wh.cache[model+blob.hash.Hex()+propsHash.Hex()]
	return str, found
}
func (wh *SAServiceWhisperCpp) addCache(model string, blob OsBlob, propsHash OsHash, value []byte) {
	wh.cache_lock.Lock()
	defer wh.cache_lock.Unlock()

	wh.cache[model+blob.hash.Hex()+propsHash.Hex()] = value
}

func (wh *SAServiceWhisperCpp) Transcribe(model string, blob OsBlob, props *SAServiceWhisperCppProps) ([]byte, float64, bool, error) {
	//find
	propsHash, err := props.Hash()
	if err != nil {
		return nil, 0, false, fmt.Errorf("Hash() failed: %w", err)
	}
	str, found := wh.FindCache(model, blob, propsHash)
	if found {
		return str, 1, true, nil
	}

	//set model
	if model != wh.last_setModel {
		err := wh.setModel(model)
		if err != nil {
			return nil, 0, false, fmt.Errorf("setModel() failed: %w", err)
		}
	}

	//translate
	out, err := wh.transcribe(blob, props)
	if err != nil {
		return nil, 0, false, fmt.Errorf("transcribe() failed: %w", err)
	}

	wh.addCache(model, blob, propsHash, out)
	return out, 1, true, nil
}

func (wh *SAServiceWhisperCpp) setModel(model string) error {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	writer.WriteField("model", "models/"+model+".bin")
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

func (wh *SAServiceWhisperCpp) transcribe(blob OsBlob, props *SAServiceWhisperCppProps) ([]byte, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	//set parameters
	{
		part, err := writer.CreateFormFile("file", "audio.wav")
		if err != nil {
			return nil, fmt.Errorf("CreateFormFile() failed: %w", err)
		}
		part.Write(blob.data)
		props.Write(writer)
	}
	writer.Close()

	req, err := http.NewRequest(http.MethodPost, wh.addr+"inference", body)
	if err != nil {
		return nil, fmt.Errorf("NewRequest() failed: %w", err)
	}
	req.Header.Add("Content-Type", writer.FormDataContentType())

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

	return resBody, nil
}
