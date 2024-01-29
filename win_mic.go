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

/*
typedef unsigned char Uint8;
void WinMic_OnAudio(void *userdata, Uint8 *stream, int length);
*/
import "C"
import (
	"sync"
	"unsafe"

	"github.com/veandco/go-sdl2/sdl"
)

type WinMic struct {
	spec   sdl.AudioSpec
	device sdl.AudioDeviceID
}

var audio_mu sync.Mutex
var audio_data []float32

//export WinMic_OnAudio
func WinMic_OnAudio(userdata unsafe.Pointer, _stream *C.Uint8, _length C.int) {
	length := int(_length) / 4 //6 because it's float32
	header := unsafe.Slice(_stream, length)
	buf := *(*[]float32)(unsafe.Pointer(&header))

	audioSamples := make([]float32, length)
	copy(audioSamples, buf)

	audio_mu.Lock()
	audio_data = append(audio_data, audioSamples...)
	audio_mu.Unlock()
}

func NewWinMic() (*WinMic, error) {
	mic := &WinMic{}

	var spec sdl.AudioSpec
	spec.Freq = 16000           //44100
	spec.Format = sdl.AUDIO_F32 //little-endian byte order
	spec.Channels = 1
	spec.Samples = 4096 //512?
	spec.Callback = sdl.AudioCallback(C.WinMic_OnAudio)
	//spec.UserData = unsafe.Pointer(mic)	//creates panic. needs to be C.malloc()

	var err error
	//defaultRecordingDeviceName := sdl.GetAudioDeviceName(0, true)
	mic.device, err = sdl.OpenAudioDevice("", true, &spec, nil, 0)
	if err != nil {
		return nil, err
	}

	mic.SetEnable(false)

	return mic, nil
}
func (mic *WinMic) Destroy() {
	sdl.CloseAudioDevice(mic.device)
}

func (mic *WinMic) SetEnable(record_now bool) {
	sdl.PauseAudioDevice(mic.device, !record_now)

	if !record_now {
		mic.Get() //clean buffer
	}
}

func (mic *WinMic) IsPlaying() bool {
	return sdl.GetAudioDeviceStatus(mic.device) == sdl.AUDIO_PLAYING
}

func (mic *WinMic) Get() []float32 {
	var ret []float32

	audio_mu.Lock()
	ret = audio_data
	audio_data = nil
	audio_mu.Unlock()

	return ret
}

/*func WinMic_test() {
	mic, err := NewWinMic()
	if err != nil {
		fmt.Println(err)
		return
	}
	defer mic.Destroy()

	mic.SetEnable(true)

	for {
		data := mic.Get()

		mx := 1 //-32767
		for _, v := range data {
			mx = OsMax(mx, int(v))
		}
		if len(data) > 0 {
			amp := float64(mx) / 32767	//bug: 32767 is for int16, not float32
			dB := 20 * math.Log10(amp)

			fmt.Println(amp, dB, mic.IsPlaying())
		}
	}
}*/

//https://github.com/veandco/go-sdl2-examples/blob/master/examples/recording-audio/recording-audio.go
