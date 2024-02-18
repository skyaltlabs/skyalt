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

import "time"

type SAServices struct {
	service_whisper_cpp *SAServiceWhisperCpp
	service_llama_cpp   *SAServiceLLamaCpp
	service_python      *SAServicePython
}

func NewSAServices() *SAServices {
	srv := &SAServices{}
	return srv
}

func (srv *SAServices) Destroy() {
	if srv.service_whisper_cpp != nil {
		srv.service_whisper_cpp.Destroy()
	}
	if srv.service_llama_cpp != nil {
		srv.service_llama_cpp.Destroy()
	}
	if srv.service_python != nil {
		srv.service_python.Destroy()
	}
}

func (srv *SAServices) GetWhisper() *SAServiceWhisperCpp {
	if srv.service_whisper_cpp == nil {
		srv.service_whisper_cpp = NewSAServiceWhisperCpp("http://127.0.0.1", "8090")
		time.Sleep(200 * time.Millisecond)
	}
	return srv.service_whisper_cpp
}

func (srv *SAServices) GetLLama() *SAServiceLLamaCpp {
	if srv.service_llama_cpp == nil {
		srv.service_llama_cpp = NewSAServiceLLamaCpp("http://127.0.0.1", "8091")
	}
	return srv.service_llama_cpp
}

func (srv *SAServices) GetPython() *SAServicePython {
	if srv.service_python == nil {
		srv.service_python = NewSAServicePython("http://127.0.0.1", "8092")
		time.Sleep(200 * time.Millisecond)
	}
	return srv.service_python
}

func (srv *SAServices) Render() {
	//dialog layout ...
	//running on/off
	//how long it's running
	//number of requests
	//avg. request time
}

func (srv *SAServices) Tick() {
	/*if srv.service_whisper_cpp != nil {
		//...
	}
	if srv.service_llama_cpp != nil {
		//...
	}
	if srv.service_python != nil {
		//...
	}*/
}
