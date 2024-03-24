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
	"net/url"
	"path/filepath"
	"strconv"
	"sync"
)

type SAServices struct {
	ui     *Ui
	port   int
	server *http.Server

	online bool

	whisperCpp *SAServiceWhisperCpp
	llamaCpp   *SAServiceLLamaCpp
	oai        *SAServiceOpenAI
	net        *SAServiceNet

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
	if srv.whisperCpp != nil {
		srv.whisperCpp.Destroy()
	}
	if srv.llamaCpp != nil {
		srv.llamaCpp.Destroy()
	}
	if srv.oai != nil {
		srv.oai.Destroy()
	}
	if srv.net != nil {
		srv.net.Destroy()
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
	if srv.whisperCpp == nil {
		srv.whisperCpp, err = NewSAServiceWhisperCpp(srv, "http://127.0.0.1", "8090", init_model)
	}
	return srv.whisperCpp, err
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

	if srv.oai == nil {
		srv.oai = NewSAServiceOpenAI(srv)
	}
	return srv.oai, nil
}

func (srv *SAServices) GetDownloader() (*SAServiceNet, error) {
	if !srv.online {
		return nil, fmt.Errorf("internet is disabled(Menu:Settings:Internet Connection)")
	}

	if srv.net == nil {
		srv.net = NewSAServiceNet(srv)
	}
	return srv.net, nil
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
	node := srv.job_app.root.FindNodeSplit(st.Node)
	if node == nil {
		http.Error(w, "Node not found", http.StatusInternalServerError)
		return
	}
	if !node.IsTypeWhispercpp() {
		http.Error(w, "Node is not type 'whispercpp'", http.StatusInternalServerError)
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

	job := wh.AddJob(srv.job_app, model, InitOsBlob(st.Data), &props)
	defer wh.RemoveJob(job)

	answer, err := job.Run()
	if err != nil {
		http.Error(w, "Complete() failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write(answer)
}

func (srv *SAServices) _prepareMessages(body []byte) ([]SAServiceMsg, *SANode, error) {

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
	node := srv.job_app.root.FindNodeSplit(st.Node)
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
	msgs, node, err := srv._prepareMessages(body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !node.IsTypeLLamacpp() {
		http.Error(w, "Node is not type 'llamacpp'", http.StatusInternalServerError)
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

	job := lm.AddJob(srv.job_app, &props)
	defer lm.RemoveJob(job)

	answer, err := job.Run()
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
	msgs, node, err := srv._prepareMessages(body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !node.IsTypeOpenAI() {
		http.Error(w, "Node is not type 'openai'", http.StatusInternalServerError)
		return
	}

	props := &SAServiceOpenAIProps{Model: node.GetAttrString("model", "gpt-3.5-turbo"), Messages: msgs}
	//more properties ..........

	//complete
	oai, err := srv.GetOpenAI()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	job := oai.AddJob(srv.job_app, props)
	defer oai.RemoveJob(job)

	answer, err := job.Run() //slots or one at the time => lock ............................
	if err != nil {
		http.Error(w, "Complete() failed: "+err.Error(), http.StatusInternalServerError)
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
		Node      string `json:"node"`
		File_path string `json:"file_path"`

		Url string `json:"url"`
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
	node := srv.job_app.root.FindNodeSplit(st.Node)
	if node == nil {
		http.Error(w, "Node not found", http.StatusInternalServerError)
		return
	}
	if !node.IsTypeNet() {
		http.Error(w, "Node is not type 'net'", http.StatusInternalServerError)
		return
	}

	//build properties
	down, err := srv.GetDownloader()
	if err != nil {
		http.Error(w, "GetDownloader() failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	u, err := url.Parse(node.GetAttrString("url", ""))
	if err != nil {
		http.Error(w, "Parse() failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	u.Path = filepath.Join(u.Path, st.Url)

	//err := net.DownloadFile(filePath, net.Url+db___model.Label+".bin") .................................

	job := down.AddJob(srv.job_app, st.File_path, u.String()) //file or RAM? ...
	defer down.RemoveJob(job)

	//check if file already exist or wait for confirmation? ........ if size is over 10MB? ...

	err = job.Run()
	if err != nil {
		http.Error(w, "Complete() failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	//w.Write(outJs)	//.....
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

func (srv *SAServices) IsJobRunning(app *SAApp) bool {

	if srv.net != nil && srv.net.FindJob(app) != nil {
		return true
	}

	if srv.llamaCpp != nil && srv.llamaCpp.FindJob(app) != nil {
		return true
	}

	if srv.oai != nil && srv.oai.FindJob(app) != nil {
		return true
	}

	if srv.whisperCpp != nil && srv.whisperCpp.FindJob(app) != nil {
		return true
	}

	return false
}

func (srv *SAServices) RenderJobs(app *SAApp) {
	ui := srv.ui

	//net
	if srv.net != nil {
		if job := srv.net.FindJob(app); job != nil {
			ui.Div_colMax(0, 100)
			ui.Div_colMax(1, 5)
			ui.Div_colMax(2, 5)
			ui.Div_colMax(3, 100)

			ui.Comp_text(1, 0, 2, 1, fmt.Sprintf("Downloading: %s", job.url), 1)
			ui.Comp_text(1, 1, 2, 1, job.GetStats(), 1)

			if ui.Comp_button(1, 3, 1, 1, "Stop", "", true) > 0 {
				job.close = true
			}

			if ui.Comp_button(2, 3, 1, 1, "Stop & Delete file", "", true) > 0 {
				job.close_and_delete = true
			}
		}
	}

	//llama
	if srv.llamaCpp != nil {
		if job := srv.llamaCpp.FindJob(app); job != nil {
			ui.Div_colMax(0, 100)
			ui.Div_colMax(1, 20)
			ui.Div_colMax(2, 100)
			ui.Div_rowMax(1, 10)

			ui.Comp_text(1, 0, 1, 1, "LLama is answering", 0)
			ui.Comp_textSelectMulti(1, 1, 1, 1, job.answer, OsV2{0, 0}, true, true, false)

			if ui.Comp_button(1, 2, 1, 1, "Cancel", "", true) > 0 {
				job.close = true
			}
		}
	}

	//openai
	if srv.oai != nil {
		if job := srv.oai.FindJob(app); job != nil {
			ui.Div_colMax(0, 100)
			ui.Div_colMax(1, 20)
			ui.Div_colMax(2, 100)
			ui.Div_rowMax(1, 10)

			ui.Comp_text(1, 0, 1, 1, "OpenAI is answering ...", 0)
			ui.Comp_textSelectMulti(1, 1, 1, 1, job.answer, OsV2{0, 0}, true, true, false)

			if ui.Comp_button(1, 2, 1, 1, "Cancel", "", true) > 0 {
				job.close = true
			}
		}
	}

	//whisper
	if srv.whisperCpp != nil {
		if job := srv.whisperCpp.FindJob(app); job != nil {
			ui.Div_colMax(0, 100)
			ui.Div_colMax(1, 20)
			ui.Div_colMax(2, 100)
			ui.Div_rowMax(1, 10)

			ui.Comp_text(1, 0, 1, 1, "Whisper is transcribing ...", 0)

			if ui.Comp_button(1, 2, 1, 1, "Cancel", "", true) > 0 {
				job.close = true
			}
		}
	}
}
