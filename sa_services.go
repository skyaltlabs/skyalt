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
	"strings"
	"time"
)

type SAServices struct {
	base   *SABase
	port   int
	server *http.Server

	online bool
}

func NewSAServices(base *SABase) *SAServices {
	srv := &SAServices{base: base}
	srv.port = 8080

	srv.Run(srv.port)
	return srv
}

func (srv *SAServices) Destroy() {
	srv.server.Shutdown(context.Background())
}

func _SAServices_getAuthID(r *http.Request) (string, error) {
	auth := r.Header.Get("Authorization")

	var found bool
	auth, found = strings.CutPrefix(auth, "Bearer ")
	if !found {
		return "", fmt.Errorf("must start with 'Bearer'")
	}

	return auth, nil
}

//handler funcs access k 1st thread struct: FindNodeSplit() .......................................
//- make copies of nodes(whisper, llama, net) into SAJobExe ...

func (srv *SAServices) handlerGetJob(w http.ResponseWriter, r *http.Request) {
	job_id, err := _SAServices_getAuthID(r)
	if err != nil {
		http.Error(w, "Auth: "+err.Error(), http.StatusInternalServerError)
		return
	}
	jb := srv.base.jobs.FindJobExe(job_id)
	if jb == nil {
		http.Error(w, "exe job not found", http.StatusInternalServerError)
		return
	}

	_, err = io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
		return
	}

	w.Write(jb.input)
}

func (srv *SAServices) handlerReturnResult(w http.ResponseWriter, r *http.Request) {
	job_id, err := _SAServices_getAuthID(r)
	if err != nil {
		http.Error(w, "Auth: "+err.Error(), http.StatusInternalServerError)
		return
	}
	jb := srv.base.jobs.FindJobExe(job_id)
	if jb == nil {
		http.Error(w, "exe job not found", http.StatusInternalServerError)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
		return
	}

	jb.outJs = body

	w.Write([]byte("{}"))
}

func (srv *SAServices) handlerReturnError(w http.ResponseWriter, r *http.Request) {
	job_id, err := _SAServices_getAuthID(r)
	if err != nil {
		http.Error(w, "Auth: "+err.Error(), http.StatusInternalServerError)
		return
	}
	jb := srv.base.jobs.FindJobExe(job_id)
	if jb == nil {
		http.Error(w, "exe job not found", http.StatusInternalServerError)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
		return
	}

	jb.outErr = errors.New(string(body))

	w.Write([]byte("{}"))
}

func (srv *SAServices) handlerWhisper(w http.ResponseWriter, r *http.Request) {
	job_id, err := _SAServices_getAuthID(r)
	if err != nil {
		http.Error(w, "Auth: "+err.Error(), http.StatusInternalServerError)
		return
	}
	jb := srv.base.jobs.FindJobExe(job_id)
	if jb == nil {
		http.Error(w, "exe job not found", http.StatusInternalServerError)
		return
	}

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
	node := NewSANodePathFromString(st.Node).Find(jb.app.root)
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

	//run & wait
	model := node.GetAttrString("model", "")
	jbw := srv.base.jobs.AddWhisper(node.app, NewSANodePath(node), model, InitOsBlob(st.Data), &props)
	for !jbw.done.Load() {
		time.Sleep(10 * time.Millisecond)
	}

	w.Write(jbw.output)
}

func (srv *SAServices) _prepareMessages(jb *SAJobExe, body []byte) ([]SAServiceMsg, *SANode, error) {
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
	node := NewSANodePathFromString(st.Node).Find(jb.app.root)
	if node == nil {
		return nil, nil, fmt.Errorf("node '%s' not found", st.Node)
	}

	return st.Messages, node, nil
}

func (srv *SAServices) handlerLLama(w http.ResponseWriter, r *http.Request) {
	job_id, err := _SAServices_getAuthID(r)
	if err != nil {
		http.Error(w, "Auth: "+err.Error(), http.StatusInternalServerError)
		return
	}
	jb := srv.base.jobs.FindJobExe(job_id)
	if jb == nil {
		http.Error(w, "exe job not found", http.StatusInternalServerError)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
		return
	}

	//extract
	msgs, node, err := srv._prepareMessages(jb, body)
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

	//run & wait
	jbw := srv.base.jobs.AddLLama(node.app, NewSANodePath(node), &props)
	for !jbw.done.Load() {
		time.Sleep(10 * time.Millisecond)
	}

	w.Write(jbw.output)
}

func (srv *SAServices) handlerOpenAI(w http.ResponseWriter, r *http.Request) {
	job_id, err := _SAServices_getAuthID(r)
	if err != nil {
		http.Error(w, "Auth: "+err.Error(), http.StatusInternalServerError)
		return
	}
	jb := srv.base.jobs.FindJobExe(job_id)
	if jb == nil {
		http.Error(w, "exe job not found", http.StatusInternalServerError)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
		return
	}

	//extract
	msgs, node, err := srv._prepareMessages(jb, body)
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

	//run & wait
	jbw := srv.base.jobs.AddOpenAI(node.app, NewSANodePath(node), props)
	for !jbw.done.Load() {
		time.Sleep(10 * time.Millisecond)
	}

	w.Write(jbw.output)
}

func (srv *SAServices) handlerNetwork(w http.ResponseWriter, r *http.Request) {
	job_id, err := _SAServices_getAuthID(r)
	if err != nil {
		http.Error(w, "Auth: "+err.Error(), http.StatusInternalServerError)
		return
	}
	jb := srv.base.jobs.FindJobExe(job_id)
	if jb == nil {
		http.Error(w, "exe job not found", http.StatusInternalServerError)
		return
	}

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
	node := NewSANodePathFromString(st.Node).Find(jb.app.root)
	if node == nil {
		http.Error(w, "Node not found", http.StatusInternalServerError)
		return
	}
	if !node.IsTypeNet() {
		http.Error(w, "Node is not type 'net'", http.StatusInternalServerError)
		return
	}

	//build properties
	u, err := url.Parse(node.GetAttrString("url", ""))
	if err != nil {
		http.Error(w, "Parse() failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	u.Path = filepath.Join(u.Path, st.Url)

	//run & wait
	srv.base.jobs.AddNet(node.app, NewSANodePath(node), st.File_path, u.String())

	w.Write([]byte("{}"))
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
