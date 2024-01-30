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
	"runtime"

	whisper "github.com/ggerganov/whisper.cpp/bindings/go"
	"github.com/go-audio/wav"
)

type SAService_WhisperToken struct {
	Text       string
	Start, End int64 //ms
}

type SAService_WhisperModel struct {
	path  string
	model *whisper.Context

	//save into 'cache.json' ...
	//key can be 'model_hexHash' ...

	cache map[[OsHash_SIZE]byte]string //key = blob.hash, value = JSON

	que []OsBlob
}

func NewSAService_WhisperModel(path string) *SAService_WhisperModel {
	wm := &SAService_WhisperModel{}

	wm.path = path
	wm.cache = make(map[[32]byte]string)

	wm.model = whisper.Whisper_init(path) //BUG: prints info ...
	if wm.model == nil {
		return nil
	}

	return wm
}

func (wm *SAService_WhisperModel) Destroy() {
	wm.model.Whisper_free()
}

func (wm *SAService_WhisperModel) FindCache(blob OsBlob) (string, bool) {
	str, found := wm.cache[blob.hash.h]
	return str, found
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

	//add into cache
	wm.cache[blob.hash.h] = string(js)

	return nil
}

func (wm *SAService_WhisperModel) Main() {

	//locking ....
	for _, q := range wm.que {
		wm.process(q)
	}

}

type SAService_Whisper struct {
	models []*SAService_WhisperModel
}

func NewSAService_Whisper() *SAService_Whisper {
	wh := &SAService_Whisper{}
	return wh
}
func (wh *SAService_Whisper) Destroy() {
	for _, wm := range wh.models {
		wm.Destroy()
	}
}

func (wh *SAService_Whisper) FindOrAddModel(path string) *SAService_WhisperModel {
	//find
	for _, wm := range wh.models {
		if wm.path == path {
			return wm
		}
	}

	//add
	wm := NewSAService_WhisperModel(path)
	wh.models = append(wh.models, wm)

	return wm
}

func (wh *SAService_Whisper) Translate(model string, blob OsBlob) (string, bool, error) {
	//find or load model
	wm := wh.FindOrAddModel(model)
	if wm == nil {
		return "", false, errors.New("Model can't be load")
	}

	//find blob
	str, found := wm.FindCache(blob)
	if found {
		return str, true, nil
	}

	//add blob to que
	wm.AddQue(blob) //mutex .....

	wm.Main()

	return "", false, nil
}

/*func find_voice_down(data []float32) int {

	vad_thold := 0.6
	freq_thold := 100.0

	step := int(0.5 * 16000) //0.5 sec
	N := OsRoundUp(float64(len(data)) / float64(step))

	last_pos := -1
	for i := 0; i < N; i++ {
		var d []float32
		if i+1 < N {
			d = data[i*step : (i+1)*step]
		} else {
			d = data[i*step:] //rest
		}

		if vad_simple(d, 16000, step/2, vad_thold, freq_thold, false) {
			last_pos = i * step
			fmt.Println("Pause", float64(i*step)/16000, "sec")
		}
	}

	return last_pos
}

// voice activity detection
func vad_simple(pcmf32 []float32, sample_rate int, n_samples_last int, vad_thold float64, freq_thold float64, verbose bool) bool {
	n_samples := len(pcmf32)
	//n_samples_last := sample_rate * last_ms / 1000

	if n_samples_last >= n_samples {
		return false // not enough samples - assume no speech
	}

	if freq_thold > 0.0 {
		high_pass_filter(pcmf32, freq_thold, sample_rate)
	}

	energy_all := 0.0
	energy_last := 0.0

	for i := 0; i < n_samples; i++ {
		energy_all += math.Abs(float64(pcmf32[i]))

		if i >= n_samples-n_samples_last {
			energy_last += math.Abs(float64(pcmf32[i]))
		}
	}

	energy_all /= float64(n_samples)
	energy_last /= float64(n_samples_last)

	if verbose {
		fmt.Printf("energy_all: %f, energy_last: %f, vad_thold: %f, freq_thold: %f\n", energy_all, energy_last, vad_thold, freq_thold)
	}

	if energy_last > vad_thold*energy_all {
		return false //started talking?
	}

	return true //stoped talking?
}
func high_pass_filter(data []float32, cutoff float64, sample_rate int) {
	rc := 1.0 / (2.0 * math.Pi * cutoff)
	dt := 1.0 / float64(sample_rate)
	alpha := float32(dt / (rc + dt))

	y := data[0]
	for i := 1; i < len(data); i++ {
		y = alpha * (y + data[i] - data[i-1])
		data[i] = y
	}
}*/
