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

	"github.com/go-audio/audio"
	"github.com/veandco/go-sdl2/sdl"
)

type WinMic struct {
	spec   sdl.AudioSpec
	device sdl.AudioDeviceID
}

// int32, but values are in range of int16
var audio_data = audio.IntBuffer{Data: nil, SourceBitDepth: 16, Format: &audio.Format{NumChannels: 1, SampleRate: 44100}}

var audio_mu sync.Mutex

//export WinMic_OnAudio
func WinMic_OnAudio(userdata unsafe.Pointer, _stream *C.Uint8, _length C.int) {
	length := int(_length) / (audio_data.SourceBitDepth / 8)
	header := unsafe.Slice(_stream, length)
	src_buf := *(*[]int16)(unsafe.Pointer(&header))

	dst := make([]int, length) //int32, but values are in range of int16
	for i, v := range src_buf {
		dst[i] = int(v) //v=int16t, which is saved into int32(audio.IntBuffer is always []int32)
	}

	audio_mu.Lock()
	audio_data.Data = append(audio_data.Data, dst...)
	audio_mu.Unlock()
}

func NewWinMic() (*WinMic, error) {
	mic := &WinMic{}

	audio_data = audio.IntBuffer{Data: nil, SourceBitDepth: 16, Format: &audio.Format{NumChannels: 1, SampleRate: 44100}}

	var spec sdl.AudioSpec
	spec.Freq = int32(audio_data.Format.SampleRate)
	spec.Format = sdl.AUDIO_S16 //audio_data.SourceBitDepth!!!
	spec.Channels = uint8(audio_data.Format.NumChannels)
	spec.Samples = 4096
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

	//BUG: when mic is enabled, it has ~2s warmup, when audio has alot of noise .........

	if record_now {
		sdl.ClearQueuedAudio(mic.device) //is this useful?
	} else {
		mic.Get() //clean buffer
	}
}

func (mic *WinMic) IsPlaying() bool {
	return sdl.GetAudioDeviceStatus(mic.device) == sdl.AUDIO_PLAYING
}

func (mic *WinMic) Get() audio.IntBuffer {
	var ret audio.IntBuffer

	audio_mu.Lock()
	ret = audio_data
	audio_data.Data = nil
	audio_mu.Unlock()

	return ret
}
