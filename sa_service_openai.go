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
	"fmt"
	"net/http"
	"os"
	"sync"
)

type SAServiceMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}
type SAServiceOpenAIProps struct {
	Model    string         `json:"model"`
	Messages []SAServiceMsg `json:"messages"`
	Stream   bool           `json:"stream"`
	//.....
	//Temperature       float64 //1
	//Max_tokens        int     //256
	//Top_p             float64 //1
	//Frequency_penalty float64 //0
	//Presence_penalty  float64 //0
}

func (p *SAServiceOpenAIProps) Hash() (OsHash, error) {
	js, err := json.Marshal(p)
	if err != nil {
		return OsHash{}, err
	}
	return InitOsHash(js)
}

type SAServiceOpenAI struct {
	jobs  *SAJobs
	cache map[string][]byte //results
	lock  sync.Mutex
}

func SAServiceOpenAI_cachePath() string {
	return "services/openai.json"
}

func NewSAServiceOpenAI(jobs *SAJobs) *SAServiceOpenAI {
	oai := &SAServiceOpenAI{jobs: jobs}

	oai.cache = make(map[string][]byte)

	//load cache
	{
		js, _ := os.ReadFile(SAServiceOpenAI_cachePath())
		if len(js) > 0 {
			err := json.Unmarshal(js, &oai.cache)
			if err != nil {
				fmt.Printf("NewSAServiceOpenAI() failed: %v\n", err)
			}
		}
	}

	return oai
}
func (oai *SAServiceOpenAI) Destroy() {
	//save cache
	js, err := json.Marshal(oai.cache)
	if err == nil {
		os.WriteFile(SAServiceOpenAI_cachePath(), js, 0644)
	}
}

func (oai *SAServiceOpenAI) FindCache(propsHash OsHash) ([]byte, bool) {
	str, found := oai.cache[propsHash.Hex()]
	return str, found
}
func (oai *SAServiceOpenAI) addCache(propsHash OsHash, value []byte) {
	oai.cache[propsHash.Hex()] = value
}

func (oai *SAServiceOpenAI) Complete(props *SAServiceOpenAIProps, wip_answer *string, stop *bool) ([]byte, error) {

	oai.lock.Lock()
	defer oai.lock.Unlock()

	//find
	propsHash, err := props.Hash()
	if err != nil {
		return nil, fmt.Errorf("Hash() failed: %w", err)
	}
	str, found := oai.FindCache(propsHash)
	if found {
		return str, nil
	}

	out, err := oai.complete(props, wip_answer, stop)
	if err != nil {
		return nil, fmt.Errorf("complete() failed: %w", err)
	}

	oai.addCache(propsHash, out)
	return out, nil
}

func (oai *SAServiceOpenAI) complete(props *SAServiceOpenAIProps, wip_answer *string, stop *bool) ([]byte, error) {
	props.Stream = true

	skey := oai.jobs.base.ui.win.io.ini.OpenAI_key
	if skey == "" {
		return nil, fmt.Errorf("OpenAI API key is not set")
	}

	js, err := json.Marshal(props)
	if err != nil {
		return nil, fmt.Errorf("Marshal() failed: %w", err)
	}

	body := bytes.NewReader([]byte(js))

	req, err := http.NewRequest(http.MethodPost, "https://api.openai.com/v1/chat/completions", body)
	if err != nil {
		return nil, fmt.Errorf("NewRequest() failed: %w", err)
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+skey)
	//req.Header.Set("Accept", "text/event-stream")
	//req.Header.Set("Cache-Control", "no-cache")
	//req.Header.Set("Connection", "keep-alive")

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Do() failed: %w", err)
	}
	defer res.Body.Close()

	answer, err := SAService_parseStream(res, wip_answer, stop)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("statusCode != 200, response: %s", answer)
	}

	return answer, nil
}
