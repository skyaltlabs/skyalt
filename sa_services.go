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

type SAServices struct {
	whisper_cpp *SAServiceWhisperCpp
	llama_cpp   *SAServiceLLamaCpp
	g4f         *SAServiceG4F
	python      *SAServicePython
}

func NewSAServices() *SAServices {
	srv := &SAServices{}
	return srv
}

func (srv *SAServices) Destroy() {
	if srv.whisper_cpp != nil {
		srv.whisper_cpp.Destroy()
	}
	if srv.llama_cpp != nil {
		srv.llama_cpp.Destroy()
	}
	if srv.g4f != nil {
		srv.g4f.Destroy()
	}
	if srv.python != nil {
		srv.python.Destroy()
	}
}

func (srv *SAServices) GetWhisper() *SAServiceWhisperCpp {
	if srv.whisper_cpp == nil {
		srv.whisper_cpp = NewSAServiceWhisperCpp("http://127.0.0.1", "8090")
	}
	return srv.whisper_cpp
}

func (srv *SAServices) GetLLama() *SAServiceLLamaCpp {
	if srv.llama_cpp == nil {
		srv.llama_cpp = NewSAServiceLLamaCpp("http://127.0.0.1", "8091")
	}
	return srv.llama_cpp
}

func (srv *SAServices) GetG4F() *SAServiceG4F {
	if srv.g4f == nil {
		srv.g4f = NewSAServiceG4F("http://127.0.0.1", "8093")
	}
	return srv.g4f
}

func (srv *SAServices) GetPython() *SAServicePython {
	if srv.python == nil {
		srv.python = NewSAServicePython("http://127.0.0.1", "8092")
	}
	return srv.python
}

func (srv *SAServices) Render() {
	//dialog layout ...
	//running on/off
	//how long it's running
	//number of requests
	//avg. request time
}

func (srv *SAServices) Tick() {
	/*if srv.whisper_cpp != nil {
		//...
	}
	if srv.llama_cpp != nil {
		//...
	}
	if srv.python != nil {
		//...
	}*/
}
