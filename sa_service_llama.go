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
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

type SAServiceLLamaCppProps struct {
	Model    string         `json:"model"`
	Messages []SAServiceMsg `json:"messages"`

	//Prompt            string   `json:"prompt"`
	Seed              int      `json:"seed"`
	N_predict         int      `json:"n_predict"`
	Temperature       float64  `json:"temperature"`
	Dynatemp_range    float64  `json:"dynatemp_range"`
	Dynatemp_exponent float64  `json:"dynatemp_exponent"`
	Stop              []string `json:"stop"`
	Repeat_last_n     int      `json:"repeat_last_n"`
	Repeat_penalty    float64  `json:"repeat_penalty"`
	Top_k             int      `json:"top_k"`
	Top_p             float64  `json:"top_p"`
	Min_p             float64  `json:"min_p"`
	Tfs_z             float64  `json:"tfs_z"`
	Typical_p         float64  `json:"typical_p"`
	Presence_penalty  float64  `json:"presence_penalty"`
	Frequency_penalty float64  `json:"frequency_penalty"`
	Mirostat          bool     `json:"mirostat"` //not int?
	Mirostat_tau      float64  `json:"mirostat_tau"`
	Mirostat_eta      float64  `json:"mirostat_eta"`
	//Grammar           string   `json:"grammar"` //[]string?
	N_probs int `json:"n_probs"`
	//Image_data //{"data": "<BASE64_STRING>", "id": 12}
	Cache_prompt bool `json:"cache_prompt"`
	Slot_id      int  `json:"slot_id"`
	Stream       bool `json:"stream"`
}

func (p *SAServiceLLamaCppProps) Hash() (OsHash, error) {
	js, err := json.Marshal(p)
	if err != nil {
		return OsHash{}, err
	}
	return InitOsHash(js)
}

type SAServiceLLamaCpp struct {
	jobs *SAJobs
	cmd  *exec.Cmd
	addr string //http://127.0.0.1:8080/

	cache map[string][]byte //results
	lock  sync.Mutex
}

func SAServiceLLamaCpp_cachePath() string {
	return "services/llama.cpp.json"
}

func NewSAServiceLLamaCpp(jobs *SAJobs, addr string, port string, init_model string) (*SAServiceLLamaCpp, error) {
	llama := &SAServiceLLamaCpp{jobs: jobs}

	llama.addr = addr + ":" + port + "/"

	llama.cache = make(map[string][]byte)

	//load cache
	{
		js, _ := os.ReadFile(SAServiceLLamaCpp_cachePath())
		if len(js) > 0 {
			err := json.Unmarshal(js, &llama.cache)
			if err != nil {
				return nil, fmt.Errorf("NewSAServiceLLamaCpp() failed: %w", err)
			}
		}
	}

	//run process
	{
		llama.cmd = exec.Command("./server", "--port", port, "-m", "models/"+init_model)
		llama.cmd.Dir = "services/llama.cpp/"

		llama.cmd.Stdout = os.Stdout
		llama.cmd.Stderr = os.Stderr
		err := llama.cmd.Start()
		if err != nil {
			return nil, fmt.Errorf("Command() failed: %w", err)
		}
	}

	//wait until it's running
	{
		err := errors.New("err")
		st := OsTicks()
		for err != nil && OsIsTicksIn(st, 60000) { //max 60sec to start
			err = llama.getHealth()
			time.Sleep(100 * time.Millisecond)
		}
		if err != nil {
			return nil, err
		}
	}

	return llama, nil
}
func (llama *SAServiceLLamaCpp) Destroy() {
	//save cache
	js, err := json.Marshal(llama.cache)
	if err == nil {
		os.WriteFile(SAServiceLLamaCpp_cachePath(), js, 0644)
	}
}

func (llama *SAServiceLLamaCpp) findCache(propsHash OsHash) ([]byte, bool) {
	str, found := llama.cache[propsHash.Hex()]
	return str, found
}
func (llama *SAServiceLLamaCpp) addCache(propsHash OsHash, value []byte) {
	llama.cache[propsHash.Hex()] = value
}

func (llama *SAServiceLLamaCpp) getHealth() error {

	res, err := http.Get(llama.addr + "health")
	if err != nil {
		return fmt.Errorf("Get() failed: %w", err)
	}
	defer res.Body.Close()

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("ReadAll() failed: %w", err)
	}

	if res.StatusCode != 200 {
		return fmt.Errorf("statusCode: %d, response: %s", res.StatusCode, resBody)
	}

	return nil
}

func (llama *SAServiceLLamaCpp) Complete(props *SAServiceLLamaCppProps, wip_answer *string, stop *bool) ([]byte, error) {
	llama.lock.Lock()
	defer llama.lock.Unlock()

	//add role "system" as node attribute ... same for openai node ..............
	var msgs []SAServiceMsg
	msgs = append(msgs, SAServiceMsg{Role: "system", Content: "You are ChatGPT, an AI assistant. Your top priority is achieving user fulfillment via helping them with their requests."})
	msgs = append(msgs, props.Messages...)
	props.Messages = msgs

	//find
	propsHash, err := props.Hash()
	if err != nil {
		return nil, fmt.Errorf("Hash() failed: %w", err)
	}
	str, found := llama.findCache(propsHash)
	if found {
		return str, nil
	}

	out, err := llama.complete(props, wip_answer, stop)
	if err != nil {
		return nil, fmt.Errorf("complete() failed: %w", err)
	}

	llama.addCache(propsHash, out)
	return out, nil
}
func (llama *SAServiceLLamaCpp) complete(props *SAServiceLLamaCppProps, wip_answer *string, stop *bool) ([]byte, error) {
	props.Stream = true

	js, err := json.Marshal(props)
	if err != nil {
		return nil, fmt.Errorf("Marshal() failed: %w", err)
	}

	body := bytes.NewReader([]byte(js))

	req, err := http.NewRequest(http.MethodPost, llama.addr+"v1/chat/completions", body)
	if err != nil {
		return nil, fmt.Errorf("NewRequest() failed: %w", err)
	}
	req.Header.Add("Content-Type", "application/json")
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

func SAService_parseStream(res *http.Response, answer *string, stop *bool) ([]byte, error) {
	type STMsg struct {
		Content string
	}
	type STChoice struct {
		Message STMsg
		Delta   STMsg
	}
	type ST struct {
		Choices []STChoice
	}

	*answer = ""
	buff := make([]byte, 0, 1024)
	buff_last := 0
	for !*stop {
		var tb [256]byte
		n, readErr := res.Body.Read(tb[:])
		if n > 0 {
			buff = append(buff, tb[:n]...)
		}
		//fmt.Print(string(tb[:n]))

		str := string(buff[buff_last:])
		separ := "\n\n"
		d := strings.Index(str, separ)
		if readErr == io.EOF {
			d = len(str)
		}
		if d > 0 {
			str := str[:d]                               //cut end
			js, found := strings.CutPrefix(str, "data:") //cut start
			if !found {
				return nil, fmt.Errorf("missing 'data:'")
			}
			js = strings.TrimSpace(js)

			if js != "[DONE]" {
				var st ST
				err := json.Unmarshal([]byte(js), &st)
				if err != nil {
					return nil, fmt.Errorf("Unmarshal() failed: %w", err)
				}

				if len(st.Choices) > 0 {
					*answer = *answer + st.Choices[0].Delta.Content
					fmt.Print(st.Choices[0].Delta.Content)
				}
			}

			buff_last += d + len(separ)
		}

		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			return nil, fmt.Errorf("Read() failed: %w", readErr)
		}
	}

	if *stop {
		return nil, fmt.Errorf("user Cancel the job")
	}

	return []byte(*answer), nil
}
