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
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

type SAServiceNetJob struct {
	downloader *SAServiceNet

	app  *SAApp
	path string
	url  string

	recv_bytes  int64
	final_bytes int64

	close            bool
	close_and_delete bool
	//err   error

	stat_time float64
	stat_recv atomic.Uint64
}

var g_SAServiceNet_flagTimeout = flag.Duration("timeout", 30*time.Minute, "HTTP timeout")

func (job *SAServiceNetJob) Run() error {

	fmt.Println("Downloading", job.url, "into", job.path)

	path := job.path + ".temp"

	//prepare temp file
	flag := os.O_CREATE | os.O_WRONLY
	if OsFileExists(path) {
		flag = os.O_APPEND | os.O_WRONLY
	}
	file, err := os.OpenFile(path, flag, 0666)
	if err != nil {
		return err
	}

	//prepare client
	req, err := http.NewRequest("GET", job.url, nil)
	if err != nil {
		file.Close()
		return err
	}
	//req.Header.Set("User-Agent", "skyalt")

	//resume download
	file_bytes, err := file.Seek(0, io.SeekEnd)
	if err != nil {
		file.Close()
		return err
	}
	if file_bytes > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", file_bytes)) //https://www.w3.org/Protocols/rfc2616/rfc2616-sec14.html
	}

	//connect
	client := http.Client{
		Timeout: *g_SAServiceNet_flagTimeout,
	}
	resp, err := client.Do(req)
	if err != nil {
		file.Close()
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		file.Close()
		return errors.New(resp.Status)
	}
	job.recv_bytes = file_bytes
	job.final_bytes = file_bytes + resp.ContentLength

	// Loop
	var retErr error
	data := make([]byte, 1024*64)
	for job.downloader.services.online && !job.close && !job.close_and_delete {

		//download
		n, err := resp.Body.Read(data)
		if err != nil {
			retErr = err
			break
		}
		//save
		m, err := file.Write(data[:n])
		if err != nil {
			retErr = err
			break
		}

		job.recv_bytes += int64(m)

		job.stat_recv.Add(uint64(m))
	}

	file.Close()

	if job.recv_bytes == job.final_bytes {
		OsFileRename(path, job.path) //<name>.temp -> <name>
		retErr = nil
	} else {
		if job.close_and_delete {
			OsFileRemove(path)
		}
	}

	if job.close || job.close_and_delete {
		retErr = fmt.Errorf("downloading canceled")
	}

	return retErr
}

func (job *SAServiceNetJob) GetProcDone() float64 {
	if job.final_bytes > 0 {
		return float64(job.recv_bytes) / float64(job.final_bytes)
	}
	return 0
}

func (job *SAServiceNetJob) GetAvgRecvBytesPerSec() float64 {
	act_time := OsTime()

	old_time := job.stat_time
	bytes := job.stat_recv.Load()

	if (act_time - job.stat_time) > 3 {
		//reset
		job.stat_time = act_time
		bytes = job.stat_recv.Swap(0)
	}

	return float64(bytes) / (act_time - old_time)
}

func (job *SAServiceNetJob) GetStats() string {
	speed := job.GetAvgRecvBytesPerSec()

	remain_sec := 0
	if speed > 0 {
		remain_sec = int(float64(job.final_bytes-job.recv_bytes) / speed)
	}

	now := time.Now()
	predict := now.Add(time.Duration(remain_sec) * time.Second)
	diff := predict.Sub(now)

	return fmt.Sprintf("%.1f%%(%.1f/%.1fMB) %.3fMB/s %v", job.GetProcDone()*100, float64(job.recv_bytes)/(1024*1024), float64(job.final_bytes)/(1024*1024), speed/(1024*1024), diff)
}

type SAServiceNet struct {
	services  *SAServices
	jobs_lock sync.Mutex
	jobs      []*SAServiceNetJob
}

func NewSAServiceNet(services *SAServices) *SAServiceNet {
	net := &SAServiceNet{services: services}

	return net
}
func (net *SAServiceNet) Destroy() {
	for len(net.jobs) > 0 {
		for _, jb := range net.jobs {
			jb.close = true
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func (net *SAServiceNet) FindJob(app *SAApp) *SAServiceNetJob {
	net.jobs_lock.Lock()
	defer net.jobs_lock.Unlock()

	for _, jb := range net.jobs {
		if jb.app == app {
			return jb
		}
	}

	return nil
}

func (net *SAServiceNet) AddJob(app *SAApp, path string, url string) *SAServiceNetJob {
	net.jobs_lock.Lock()
	defer net.jobs_lock.Unlock()

	//find
	for _, jb := range net.jobs {
		if jb.url == url {
			return jb
		}
	}

	//add
	job := &SAServiceNetJob{downloader: net, app: app, path: path, url: url}
	net.jobs = append(net.jobs, job)

	return job
}

func (net *SAServiceNet) RemoveJob(job *SAServiceNetJob) bool {
	net.jobs_lock.Lock()
	defer net.jobs_lock.Unlock()

	for i, jb := range net.jobs {
		if jb == job {
			net.jobs = append(net.jobs[:i], net.jobs[i+1:]...) //remove
			return true
		}
	}

	return false
}
