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
	"os"
	"runtime"

	whisper "github.com/ggerganov/whisper.cpp/bindings/go"
	"github.com/go-audio/wav"
)

type SAService_WhisperToken struct {
	Text       string
	Start, End int64 //ms
}

type SAService_WhisperModel struct {
	parent *SAService_Whisper
	path   string
	model  *whisper.Context

	que []OsBlob
}

func NewSAService_WhisperModel(path string, parent *SAService_Whisper) *SAService_WhisperModel {
	wm := &SAService_WhisperModel{}
	wm.parent = parent
	wm.path = path

	wm.model = whisper.Whisper_init(path) //BUG: prints info ...
	if wm.model == nil {
		return nil
	}

	return wm
}

func (wm *SAService_WhisperModel) Destroy() {
	wm.model.Whisper_free()
}

func (wm *SAService_WhisperModel) AddQue(blob OsBlob) {
	wm.que = append(wm.que, blob)
}

func (wm *SAService_WhisperModel) SAService_WhisperModel_convert(blob OsBlob) ([]float32, error) {

	reader := bytes.NewReader(blob.data)

	dec := wav.NewDecoder(reader)
	buf, err := dec.FullPCMBuffer()
	if err != nil {
		return nil, err
	}

	if dec.NumChans != 1 {
		return nil, fmt.Errorf("unsupported number of channels: %d", dec.NumChans)
	}

	buff := buf.AsFloat32Buffer().Data

	if dec.SampleRate != whisper.SampleRate {
		buff = OsAudio_resample(int(dec.SampleRate), whisper.SampleRate, buff) //must be 16kHz
	}

	return buff, nil
}

func (wm *SAService_WhisperModel) process(blob OsBlob) error {

	buff, err := wm.SAService_WhisperModel_convert(blob)
	if err != nil {
		return err
	}

	params := wm.model.Whisper_full_default_params(whisper.SAMPLING_GREEDY)
	params.SetTranslate(false)
	params.SetPrintSpecial(false)
	params.SetPrintProgress(false)
	params.SetPrintRealtime(false)
	params.SetTokenTimestamps(true) //!!!
	params.SetThreads(runtime.NumCPU())
	params.SetNoContext(true)
	wm.model.Whisper_reset_timings()

	var tokens []SAService_WhisperToken

	wm.model.Whisper_full(params, buff, nil, func(new int) {
		num_segments := wm.model.Whisper_full_n_segments()
		s0 := num_segments - new
		for s := s0; s < num_segments; s++ {

			ntoks := wm.model.Whisper_full_n_tokens(s)
			//result := make([]Token, ctx.Whisper_full_n_tokens(n))
			for t := 0; t < ntoks; t++ {
				data := wm.model.Whisper_full_get_token_data(s, t)
				tokens = append(tokens, SAService_WhisperToken{Text: wm.model.Whisper_full_get_token_text(s, t), Start: data.T0() * 10, End: data.T1() * 10})
			}
		}
	}, func(progress int) {
		//fmt.Println("Progress:", progress)
	})

	js, err := json.Marshal(tokens)
	if err != nil {
		return err
	}

	wm.parent.addCache(wm.path, blob, string(js))
	return nil
}

func (wm *SAService_WhisperModel) Main() {

	//locking read que ....
	for _, q := range wm.que {
		err := wm.process(q)
		if err != nil {
			fmt.Println(err)
		}
	}

}

type SAService_Whisper struct {
	models []*SAService_WhisperModel

	cache map[string]string //key = blob.hash, value = JSON

}

func SAService_Whisper_cachePath() string {
	return "models/whisper/cache.json"
}

func NewSAService_Whisper() *SAService_Whisper {
	wh := &SAService_Whisper{}

	wh.cache = make(map[string]string)

	js, _ := os.ReadFile(SAService_Whisper_cachePath())
	if len(js) > 0 {
		err := json.Unmarshal(js, &wh.cache)
		if err != nil {
			fmt.Printf("NewSAService_Whisper() failed: %v\n", err)
		}
	}

	return wh
}
func (wh *SAService_Whisper) Destroy() {

	js, err := json.Marshal(wh.cache)
	if err == nil {
		os.WriteFile(SAService_Whisper_cachePath(), js, 0644)
	}

	for _, wm := range wh.models {
		wm.Destroy()
	}
}

func (wh *SAService_Whisper) findCache(model string, blob OsBlob) (string, bool) {
	str, found := wh.cache[model+blob.Hex()]
	return str, found
}
func (wh *SAService_Whisper) addCache(model string, blob OsBlob, value string) {
	wh.cache[model+blob.Hex()] = value
}

func (wh *SAService_Whisper) FindOrAddModel(path string) *SAService_WhisperModel {
	//find
	for _, wm := range wh.models {
		if wm.path == path {
			return wm
		}
	}

	//add
	wm := NewSAService_WhisperModel(path, wh)
	wh.models = append(wh.models, wm)

	return wm
}

func (wh *SAService_Whisper) Translate(model string, blob OsBlob) (string, float64, bool, error) {
	//find blob
	str, found := wh.findCache(model, blob)
	if found {
		return str, 1, true, nil
	}

	//find or load model
	wm := wh.FindOrAddModel(model)
	if wm == nil {
		return "", 0, false, errors.New("model can't be load")
	}

	//add blob to que
	wm.AddQue(blob) //mutex .....

	wm.Main()

	return "", 0.5, false, nil
}
