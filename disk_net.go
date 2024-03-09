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
	"fmt"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

type DiskNetJob struct {
	url   string
	agent string

	progress float64 //...
	done     bool

	data []byte
	err  error
}

func (job *DiskNetJob) downloadWithAgent() {
	client := &http.Client{}

	req, err := http.NewRequest("GET", job.url, nil)
	if err != nil {
		job.err = err
		return
	}

	req.Header.Set("User-Agent", job.agent)

	resp, err := client.Do(req)
	if err != nil {
		job.err = err
		return
	}

	job.data, err = io.ReadAll(resp.Body)
	if err != nil {
		job.err = err
		return
	}
}

func (job *DiskNetJob) download() {

	resp, err := http.Get(job.url)
	if err != nil {
		job.err = err
		return
	}

	job.data, err = io.ReadAll(resp.Body)
	if err != nil {
		job.err = err
		return
	}
}

type DiskNet struct {
	lock      sync.Mutex
	interrupt atomic.Bool

	jobs   []*DiskNetJob
	online bool

	num_jobs_done   int
	num_jobs_errors int
}

func NewDiskNet() *DiskNet {
	net := &DiskNet{}
	net.online = true

	go net.Loop()

	return net
}

func (net *DiskNet) Destroy() {
	net.interrupt.Store(true)
}

func (net *DiskNet) Loop() {

	for !net.interrupt.Load() {

		//get
		jb := net.getNext()

		//download
		if jb != nil {
			if jb.agent != "" {
				jb.downloadWithAgent()
			} else {
				jb.download()
			}
			jb.done = true

			net.num_jobs_done++
			if jb.err != nil {
				net.num_jobs_errors++
			}
		} else {
			time.Sleep(20 * time.Millisecond)
		}
	}
}

func (net *DiskNet) getNext() *DiskNetJob {
	net.lock.Lock()
	defer net.lock.Unlock()

	var found *DiskNetJob
	for _, jb := range net.jobs {
		if !jb.done {
			found = jb
			break
		}
	}
	return found
}

func (net *DiskNet) add(url string, agent string) *DiskNetJob {
	net.lock.Lock()
	defer net.lock.Unlock()

	//find
	for i, jb := range net.jobs {
		if jb.url == url && jb.agent == agent {
			if jb.done {
				net.jobs = append(net.jobs[:i], net.jobs[i+1:]...) //remove
			}
			return jb
		}
	}

	//add
	jb := &DiskNetJob{url: url, agent: agent}
	net.jobs = append(net.jobs, jb) //add
	return jb
}

func (net *DiskNet) GetFile(url string, agent string) ([]byte, bool, float64, error) {

	if net.online {
		jb := net.add(url, agent)
		return jb.data, jb.done, jb.progress, jb.err
	} else {
		return nil, true, 0, fmt.Errorf("internet is disabled(Menu:Settings:Internet Connection)")
	}
}
