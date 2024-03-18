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

type SAServiceDownloaderJob struct {
	downloader *SAServiceDownloader

	path string
	url  string

	recv_bytes  int64
	final_bytes int64

	close bool
	err   error

	stat_time float64
	stat_recv atomic.Uint64
}

type SAServiceDownloader struct {
	services *SAServices
	lock     sync.Mutex //for cache
	jobs     []*SAServiceDownloaderJob
}

func NewSAServiceDownloader(services *SAServices) *SAServiceDownloader {
	down := &SAServiceDownloader{services: services}

	return down
}
func (down *SAServiceDownloader) Destroy() {
	for len(down.jobs) > 0 {
		for _, jb := range down.jobs {
			jb.close = true
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func (down *SAServiceDownloader) AddJob(path string, url string) *SAServiceDownloaderJob {
	down.lock.Lock()
	defer down.lock.Unlock()

	//find
	for _, jb := range down.jobs {
		if jb.url == url {
			return jb
		}
	}

	//add
	job := &SAServiceDownloaderJob{downloader: down, path: path, url: url}
	down.jobs = append(down.jobs, job)

	go job.run()

	return job
}

func (down *SAServiceDownloader) RemoveJob(job *SAServiceDownloaderJob) bool {
	down.lock.Lock()
	defer down.lock.Unlock()

	for i, jb := range down.jobs {
		if jb == job {
			down.jobs = append(down.jobs[:i], down.jobs[i+1:]...) //remove
			return true
		}
	}

	return false
}

var g_SAServiceDownloader_flagTimeout = flag.Duration("timeout", 30*time.Minute, "HTTP timeout")

func (job *SAServiceDownloaderJob) run() {

	fmt.Println("Downloading", job.url, "into", job.path)

	path := job.path + ".temp"

	//prepare temp file
	flag := os.O_CREATE | os.O_WRONLY
	if OsFileExists(path) {
		flag = os.O_APPEND | os.O_WRONLY
	}
	file, err := os.OpenFile(path, flag, 0666)
	if err != nil {
		job.err = err
	}

	//prepare client
	req, err := http.NewRequest("GET", job.url, nil)
	if err != nil {
		job.err = err
		file.Close()
		return
	}
	//req.Header.Set("User-Agent", "skyalt")

	//resume download
	file_bytes, err := file.Seek(0, io.SeekEnd)
	if err != nil {
		job.err = err
		file.Close()
		return
	}
	if file_bytes > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", file_bytes)) //https://www.w3.org/Protocols/rfc2616/rfc2616-sec14.html
	}

	//connect
	client := http.Client{
		Timeout: *g_SAServiceDownloader_flagTimeout,
	}
	resp, err := client.Do(req)
	if err != nil {
		job.err = err
		file.Close()
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		job.err = errors.New(resp.Status)
		file.Close()
		return
	}
	job.final_bytes = resp.ContentLength

	// Loop
	data := make([]byte, 1024*64)
	for job.downloader.services.online && !job.close {

		//download
		n, err := resp.Body.Read(data)
		if err != nil {
			job.err = err
			break
		}
		//save
		m, err := file.Write(data[:n])
		if err != nil {
			job.err = err
			break
		}

		job.recv_bytes += int64(m)

		job.stat_recv.Add(uint64(m))

		//if job.recv_bytes == resp.ContentLength {
		//	w.Close()
		//}
	}

	file.Close()

	if job.recv_bytes == job.final_bytes {
		OsFileRename(path, job.path)
	}

	job.downloader.RemoveJob(job)
}

func (job *SAServiceDownloaderJob) GetProcDone() float64 {
	return float64(job.recv_bytes) / float64(job.final_bytes)
}

func (job *SAServiceDownloaderJob) GetAvgRecvBytesPerSec() float64 {
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

func (job *SAServiceDownloaderJob) GetStats() string {
	speed := job.GetAvgRecvBytesPerSec()

	remain_sec := 0
	if speed > 0 {
		remain_sec = int(float64(job.final_bytes-job.recv_bytes) / speed)
	}

	now := time.Now()
	predict := now.Add(time.Duration(remain_sec) * time.Second)
	diff := predict.Sub(now)

	return fmt.Sprintf("%.1f%% %.3fMB/s %v", job.GetProcDone()*100, speed, diff)
}
