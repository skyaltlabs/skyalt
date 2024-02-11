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
	"flag"
	"fmt"
	"net/http"
	"os"
	"sync/atomic"
	"time"
)

type SAJob struct {
	interrupt atomic.Bool
	done      atomic.Bool

	node     SANodePath
	run_time float64

	err           error
	progress      float64
	progress_desc string
}

func NewSAJob(node *SANode) *SAJob {
	return &SAJob{node: NewSANodePath(node), progress: 0.001}
}
func (jb *SAJob) Interrupt() {
	jb.interrupt.Store(true)
}
func (jb *SAJob) Done(start_time float64) {
	jb.run_time = OsTime() - start_time
	jb.done.Store(true)
}

type SAJobs struct {
	app  *SAApp
	jobs []*SAJob
}

func NewSAJobs(app *SAApp) *SAJobs {
	return &SAJobs{app: app}
}

func (jbs *SAJobs) StopAll() {

	for _, jb := range jbs.jobs {
		jb.Interrupt()
	}

	running := true
	for running {
		running = false
		for _, jb := range jbs.jobs {
			if !jb.done.Load() {
				running = true
			}
		}
		time.Sleep(50 * time.Millisecond)
	}

	jbs.jobs = nil
}

func (jbs *SAJobs) Destroy() {
	jbs.StopAll()
}

func (jbs *SAJobs) Num() int {
	return len(jbs.jobs)
}

func (jbs *SAJobs) Tick(enableExe bool) {

	if jbs.app.root != nil {
		jbs.app.root.ResetProgress()
	}

	if !enableExe {
		jbs.StopAll()
	}

	for i := len(jbs.jobs) - 1; i >= 0; i-- {
		jb := jbs.jobs[i]

		//update
		nd := jb.node.FindPath(jbs.app.root)
		nd.errExe = jb.err
		nd.progress_desc = jb.progress_desc
		nd.progress = jb.progress

		//remove done
		if jb.done.Load() {
			nd.exeTimeSec = jb.run_time
			jbs.jobs = append(jbs.jobs[:i], jbs.jobs[i+1:]...) //remove
		}
	}
}

func (jbs *SAJobs) AddJob(job *SAJob) {

	//find and cancel previous ....

	jbs.jobs = append(jbs.jobs, job)
}

var flagTimeout = flag.Duration("timeout", 30*time.Minute, "HTTP timeout")

func SAJob_NN_whisper_cpp_downloader(job *SAJob, url string, dst string, modelName string) {

	fmt.Println("Downloading", url, "into", dst)
	defer job.Done(OsTime())

	client := http.Client{
		Timeout: *flagTimeout,
	}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		job.err = err
		return
	}

	//req.Header.Set("User-Agent", "skyalt")
	resp, err := client.Do(req)
	if err != nil {
		job.err = err
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		job.err = fmt.Errorf("%s: %s", url, resp.Status)
		return
	}

	// file already exists(same size)
	if info, err := os.Stat(dst); err == nil && info.Size() == resp.ContentLength {
		fmt.Println("Skipping", dst, "as it already exists")
		return
	}

	w, err := os.Create(dst)
	if err != nil {
		job.err = err
		return
	}
	defer w.Close()

	job.progress_desc = fmt.Sprint("Downloading: ", modelName)

	// Loop
	data := make([]byte, 1024*64)
	recv_bytes := int64(0)
	ticker := time.NewTicker(1 * time.Second)
	for !job.interrupt.Load() {
		select {
		case <-ticker.C:
			job.progress = float64(recv_bytes) / float64(resp.ContentLength)
		default:
			//download
			n, err := resp.Body.Read(data)
			if err != nil {
				job.err = err
				w.Close()
				return
			}
			//save
			m, err := w.Write(data[:n])
			if err != nil {
				job.err = err
				w.Close()
				return
			}

			recv_bytes += int64(m)
			job.progress = float64(recv_bytes) / float64(resp.ContentLength)
		}
	}

	if recv_bytes < resp.ContentLength {
		w.Close()
		OsFileRemove(dst)
	}
}
