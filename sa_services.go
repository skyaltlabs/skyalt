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
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
)

type SAServices struct {
	ui     *Ui
	port   int
	server *http.Server

	online bool

	whisper_cpp *SAServiceWhisperCpp
	llamaCpp    *SAServiceLLamaCpp
	openai      *SAServiceOpenAI
	downloader  *SAServiceDownloader

	job_str       []byte
	job_app       *SAApp
	job_result    []byte
	job_result_wg sync.WaitGroup
	job_err       error
}

func NewSAServices(ui *Ui) *SAServices {
	srv := &SAServices{ui: ui}
	srv.port = 8080

	srv.Run(srv.port)
	return srv
}

func (srv *SAServices) Destroy() {
	if srv.whisper_cpp != nil {
		srv.whisper_cpp.Destroy()
	}
	if srv.llamaCpp != nil {
		srv.llamaCpp.Destroy()
	}
	if srv.openai != nil {
		srv.openai.Destroy()
	}
	if srv.downloader != nil {
		srv.downloader.Destroy()
	}

	srv.server.Shutdown(context.Background())
}

func (srv *SAServices) SetJob(job []byte, app *SAApp) {
	srv.job_str = job
	srv.job_app = app
	srv.job_err = nil
	srv.job_result_wg.Add(1)
}

func (srv *SAServices) GetResult() ([]byte, error) {
	srv.job_result_wg.Wait()
	return srv.job_result, srv.job_err
}

func (srv *SAServices) GetWhisper(init_model string) (*SAServiceWhisperCpp, error) {
	var err error
	if srv.whisper_cpp == nil {
		srv.whisper_cpp, err = NewSAServiceWhisperCpp(srv, "http://127.0.0.1", "8090", init_model)
	}
	return srv.whisper_cpp, err
}

func (srv *SAServices) GetLLama(init_model string) (*SAServiceLLamaCpp, error) {
	var err error
	if srv.llamaCpp == nil {
		srv.llamaCpp, err = NewSAServiceLLamaCpp(srv, "http://127.0.0.1", "8091", init_model)
	}
	return srv.llamaCpp, err
}

func (srv *SAServices) GetOpenAI() (*SAServiceOpenAI, error) {
	if !srv.online {
		return nil, fmt.Errorf("internet is disabled(Menu:Settings:Internet Connection)")
	}

	if srv.openai == nil {
		srv.openai = NewSAServiceOpenAI(srv)
	}
	return srv.openai, nil
}

func (srv *SAServices) GetDownloader() (*SAServiceDownloader, error) {
	if !srv.online {
		return nil, fmt.Errorf("internet is disabled(Menu:Settings:Internet Connection)")
	}

	if srv.downloader == nil {
		srv.downloader = NewSAServiceDownloader(srv)
	}
	return srv.downloader, nil
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
	if srv.llamaCpp != nil {
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

func (srv *SAServices) handlerReturnResult(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
		return
	}

	srv.job_result = body
	srv.job_err = nil
	srv.job_result_wg.Done()

	w.Write([]byte("{}"))
}
func (srv *SAServices) handlerReturnError(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
		return
	}

	srv.job_result = nil
	srv.job_err = errors.New(string(body))
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
	model := node.GetAttrString("model", "")
	wh, err := srv.GetWhisper(model)
	if err != nil {
		http.Error(w, "GetWhisper() failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	outJs, _, _, err := wh.Translate(model, InitOsBlob(st.Data), &props)
	if err != nil {
		http.Error(w, "Translate() failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write(outJs)
}

func (srv *SAServices) _extractMessages(body []byte) ([]SAServiceMsg, *SANode, error) {

	//get base struct
	type St struct {
		Node     string         `json:"node"`
		Messages []SAServiceMsg `json:"messages"`
	}
	var st St
	err := json.Unmarshal(body, &st)
	if err != nil {
		return nil, nil, fmt.Errorf("Unmarshal() failed: %w", err)
	}

	//find node
	if srv.job_app == nil {
		return nil, nil, fmt.Errorf("job_app failed: %w", err)
	}
	node := srv.job_app.root.FindNode(st.Node)
	if node == nil {
		return nil, nil, fmt.Errorf("node '%s' not found", st.Node)
	}

	return st.Messages, node, nil
}

func (srv *SAServices) handlerLLama(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
		return
	}

	//extract
	msgs, node, err := srv._extractMessages(body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// get llama properties from Node
	js, err := json.Marshal(node.Attrs)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var props SAServiceLLamaCppProps
	err = json.Unmarshal(js, &props)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	//add Model and Messages into properties
	props.Model = node.GetAttrString("model", "")
	props.Messages = msgs

	//complete
	lm, err := srv.GetLLama(props.Model)
	if err != nil {
		http.Error(w, "GetLLama() failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	answer, err := lm.Complete(&props)
	if err != nil {
		http.Error(w, "Complete() failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write(answer)
}

func (srv *SAServices) handlerOpenAI(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
		return
	}

	//extract
	msgs, node, err := srv._extractMessages(body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	//complete
	g4f, err := srv.GetOpenAI()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	answer, err := g4f.Complete(&SAServiceOpenAIProps{Model: node.GetAttrString("model", "gpt-4-turbo-preview"), Messages: msgs}) //more properties ..........
	if err != nil {
		http.Error(w, "OpenAI() failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write(answer)
}

func (srv *SAServices) handlerNetwork(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
		return
	}

	//get base struct
	type Net struct {
		Node string `json:"node"`
		Path string `json:"path"` //otherwise return []byte(keep it in RAM)
	}
	var st Net
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
	/*down, err := srv.GetDownloader()
	if err != nil {
		http.Error(w, "GetDownloader() failed: "+err.Error(), http.StatusInternalServerError)
		return
	}*/

	//down.AddJob()
	//......................
	//struct: file or RAM?

	//user can cancel ... bottom header with actual action
	//ask: start from scratch or continue - if size is over 10MB?

	//down.AddJob()

	//w.Write(outJs)
}

func (srv *SAServices) Run(port int) {
	mux := http.NewServeMux()
	mux.HandleFunc("/getjob", srv.handlerGetJob)
	mux.HandleFunc("/returnresult", srv.handlerReturnResult)
	mux.HandleFunc("/returnerror", srv.handlerReturnError)
	mux.HandleFunc("/whispercpp", srv.handlerWhisper)
	mux.HandleFunc("/llamacpp", srv.handlerLLama)
	mux.HandleFunc("/openai", srv.handlerOpenAI)
	mux.HandleFunc("/net", srv.handlerNetwork)
	srv.server = &http.Server{Addr: ":" + strconv.Itoa(port), Handler: mux}

	go func() {
		err := srv.server.ListenAndServe()
		if err != nil {
			return
		}
	}()
}
