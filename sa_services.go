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
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"sync"
)

type SAServices struct {
	port   int
	server *http.Server

	whisper_cpp *SAServiceWhisperCpp
	llama_cpp   *SAServiceLLamaCpp
	g4f         *SAServiceG4F

	job_str       []byte
	job_app       *SAApp
	job_result    []byte
	job_result_wg sync.WaitGroup
}

func NewSAServices() *SAServices {
	srv := &SAServices{}
	srv.port = 8080

	srv.Run(srv.port)
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

	srv.server.Shutdown(context.Background())
}

func (srv *SAServices) SetJob(job []byte, app *SAApp) {
	srv.job_str = job
	srv.job_app = app
	srv.job_result_wg.Add(1)
}

func (srv *SAServices) GetResult() []byte {
	srv.job_result_wg.Wait()
	return srv.job_result
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
	}*/
}

func (srv *SAServices) handlerGetJob(w http.ResponseWriter, r *http.Request) {
	_, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
		return
	}

	w.Write(srv.job_str)
}

func (srv *SAServices) handlerSetResult(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
		return
	}

	srv.job_result = body
	srv.job_result_wg.Done()

	w.Write([]byte("{}"))
}

func (srv *SAServices) handlerWhisper(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
		return
	}

	//get base struct
	type Whisper_cpp struct {
		Node      string `json:"node"`
		File_path string `json:"file_path"`
		Data      []byte `json:"data"`
	}
	var st Whisper_cpp
	err = json.Unmarshal(body, &st)
	if err != nil {
		http.Error(w, "Unmarshal() 1 failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	//find node
	if srv.job_app == nil {
		http.Error(w, "job_app is nil", http.StatusInternalServerError)
		return
	}
	node := srv.job_app.root.FindNode(st.Node)
	if node == nil {
		http.Error(w, "Node not found", http.StatusInternalServerError)
		return
	}

	//build properties
	propsJs, err := json.Marshal(node.Attrs)
	if err != nil {
		http.Error(w, "Marshal() failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	var props SAServiceWhisperCppProps
	err = json.Unmarshal(propsJs, &props)
	if err != nil {
		http.Error(w, "Unmarshal() 2 failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	//translate
	outJs, _, _, err := srv.GetWhisper().Translate(node.GetAttrString("model", ""), InitOsBlob(st.Data), &props)
	if err != nil {
		http.Error(w, "Whisper() failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write(outJs)
}

func (srv *SAServices) handlerLLama(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
		return
	}

	var res []byte
	res = body
	//...

	w.Write(res)
}

func (srv *SAServices) handlerG4F(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
		return
	}

	//get base struct
	type G4fMessage struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	type G4f struct {
		Node     string       `json:"node"`
		Messages []G4fMessage `json:"messages"`
	}
	var st G4f
	err = json.Unmarshal(body, &st)
	if err != nil {
		http.Error(w, "Unmarshal() 1 failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	//find node
	if srv.job_app == nil {
		http.Error(w, "job_app is nil", http.StatusInternalServerError)
		return
	}
	node := srv.job_app.root.FindNode(st.Node)
	if node == nil {
		http.Error(w, "Node not found", http.StatusInternalServerError)
		return
	}

	// build messages
	messagesJs, err := json.Marshal(node.Attrs)
	if err != nil {
		http.Error(w, "Marshal() failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	//complete
	answer, err := srv.GetG4F().Complete(&SAServiceG4FProps{Model: "gpt-4-turbo", Messages: string(messagesJs)})
	if err != nil {
		http.Error(w, "G4F() failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write(answer)
}
func (srv *SAServices) Run(port int) {
	mux := http.NewServeMux()
	mux.HandleFunc("/getjob", srv.handlerGetJob)
	mux.HandleFunc("/setresult", srv.handlerSetResult)
	mux.HandleFunc("/whisper", srv.handlerWhisper)
	mux.HandleFunc("/llama", srv.handlerLLama)
	mux.HandleFunc("/g4f", srv.handlerG4F)
	srv.server = &http.Server{Addr: ":" + strconv.Itoa(port), Handler: mux}

	go func() {
		err := srv.server.ListenAndServe()
		if err != nil {
			return
		}
	}()
}
