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
	"sync"
	"time"
)

// Doc: https://help.openai.com/en/articles/7042661-moving-from-completions-to-chat-completions-in-the-openai-api
type SAServiceG4FMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}
type SAServiceG4FProps struct {
	Model    string            `json:"model"`
	Messages []SAServiceG4FMsg `json:"messages"`
}

func (p *SAServiceG4FProps) Hash() (OsHash, error) {
	js, err := json.Marshal(p)
	if err != nil {
		return OsHash{}, err
	}
	return InitOsHash(js)
}

type SAServiceG4F struct {
	cmd  *exec.Cmd
	addr string //http://127.0.0.1:8080/

	cache      map[string][]byte //results
	cache_lock sync.Mutex        //for cache
}

func SAServiceG4F_cachePath() string {
	return "services/g4f.json"
}

func NewSAServiceG4F(addr string, port string) *SAServiceG4F {
	wh := &SAServiceG4F{}

	wh.addr = addr + ":" + port + "/"

	wh.cache = make(map[string][]byte)

	//load cache
	{
		js, _ := os.ReadFile(SAServiceG4F_cachePath())
		if len(js) > 0 {
			err := json.Unmarshal(js, &wh.cache)
			if err != nil {
				fmt.Printf("NewSAServiceG4F() failed: %v\n", err)
			}
		}
	}

	//run process
	{
		wh.cmd = exec.Command("python3", "services/g4f/server.py", port)
		wh.cmd.Stdout = os.Stdout
		wh.cmd.Stderr = os.Stderr
		err := wh.cmd.Start()
		if err != nil {
			fmt.Println(err)
		}
	}

	//wait until it's running
	{
		err := errors.New("err")
		st := OsTicks()
		for err != nil && OsIsTicksIn(st, 3000) {
			_, err = wh.complete(&SAServiceG4FProps{})
			time.Sleep(50 * time.Millisecond)
		}
	}

	return wh
}
func (wh *SAServiceG4F) Destroy() {
	//save cache
	js, err := json.Marshal(wh.cache)
	if err == nil {
		os.WriteFile(SAServiceG4F_cachePath(), js, 0644)
	}
}

func (wh *SAServiceG4F) FindCache(propsHash OsHash) ([]byte, bool) {
	wh.cache_lock.Lock()
	defer wh.cache_lock.Unlock()

	str, found := wh.cache[propsHash.Hex()]
	return str, found
}
func (wh *SAServiceG4F) addCache(propsHash OsHash, value []byte) {
	wh.cache_lock.Lock()
	defer wh.cache_lock.Unlock()

	wh.cache[propsHash.Hex()] = value
}

func (wh *SAServiceG4F) Complete(props *SAServiceG4FProps) ([]byte, error) {
	//find
	propsHash, err := props.Hash()
	if err != nil {
		return nil, fmt.Errorf("Hash() failed: %w", err)
	}
	str, found := wh.FindCache(propsHash)
	if found {
		return str, nil
	}

	out, err := wh.complete(props)
	if err != nil {
		return nil, fmt.Errorf("complete() failed: %w", err)
	}

	wh.addCache(propsHash, out)
	return out, nil
}

func (wh *SAServiceG4F) complete(props *SAServiceG4FProps) ([]byte, error) {
	js, err := json.Marshal(props)
	if err != nil {
		return nil, fmt.Errorf("Marshal() failed: %w", err)
	}

	body := bytes.NewReader([]byte(js))

	req, err := http.NewRequest(http.MethodPost, wh.addr+"completion", body)
	if err != nil {
		return nil, fmt.Errorf("NewRequest() failed: %w", err)
	}
	req.Header.Add("Content-Type", "application/json")

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Do() failed: %w", err)
	}

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("ReadAll() failed: %w", err)
	}

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("statusCode != 200, response: %s", resBody)
	}

	var str string
	err = json.Unmarshal(resBody, &str)
	if err != nil {
		return nil, fmt.Errorf("Unmarshal() failed: %w", err)
	}

	return []byte(str), nil
}
