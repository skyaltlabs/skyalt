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
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

type SAJobTimeStat struct {
	times    []float64
	time_avg float64
}

func InitSAJobStats(init_val float64) SAJobTimeStat {
	var st SAJobTimeStat
	st.Add(init_val) //1sec
	return st
}

func (st *SAJobTimeStat) Add(tm_sec float64) {
	//add
	if len(st.times) > 10 {
		st.times = st.times[1:] //cut one
	}
	st.times = append(st.times, tm_sec) //add

	//update avg
	sum := 0.0
	for _, tm := range st.times {
		sum += tm
	}
	st.time_avg = sum / float64(len(st.times))
}

type SAJobCompile struct {
	jobs    *SAJobs
	st_time float64

	app      *SAApp
	node     SANodePath
	dirPath  string //temp/go/
	fileName string //xzy.go

	output []byte
	outErr error

	dt_time float64
	done    atomic.Bool
}

func NewSAJobCompile(app *SAApp, node SANodePath, dirPath string, fileName string, jobs *SAJobs) *SAJobCompile {
	jb := &SAJobCompile{jobs: jobs, st_time: OsTime()}

	jb.app = app
	jb.node = node
	jb.dirPath = dirPath
	jb.fileName = fileName

	return jb
}

func (jb *SAJobCompile) Run() {
	defer jb.done.Store(true)

	cmd := exec.Command("go", "build", jb.fileName)
	cmd.Dir = jb.dirPath

	var err error
	jb.output, err = cmd.CombinedOutput()
	if err != nil {
		jb.outErr = errors.New(string(jb.output))
		jb.output = nil
	}

	jb.dt_time = OsTime() - jb.st_time
}

func (jb *SAJobCompile) GetProgress() (string, float64) {
	dt := OsTime() - jb.st_time
	return fmt.Sprintf("Compiling %s", jb.fileName), dt / jb.jobs.compile_stats.time_avg
}

func (jb *SAJobCompile) RenderProgress(y *int) bool {
	ui := jb.jobs.base.ui

	str, proc := jb.GetProgress()
	ui.Comp_text(0, *y, 1, 1, fmt.Sprintf("%s ... %.1f%%", str, proc*100), 0)
	(*y)++
	return true
}

func (jb *SAJobCompile) PostRun() {
	node := jb.node.FindPath(jb.app.root)
	if node == nil {
		fmt.Printf("Warning: SAJobCompile node '%s' not found\n", jb.node.String())
		return
	}

	node.Code.file_err = jb.outErr

	fmt.Printf("SAJobCompile '%s' finished in %f\n", jb.fileName, jb.dt_time)
}

type SAJobExe struct {
	jobs    *SAJobs
	st_time float64

	job_id      string
	app         *SAApp
	node        SANodePath
	dirPath     string //temp/go/
	programName string //xzy

	input []byte

	outJs  []byte
	outCmd []byte
	outErr error

	dt_time float64
	done    atomic.Bool
}

func NewSAJobExe(job_id string, app *SAApp, node SANodePath, dirPath string, programName string, input []byte, jobs *SAJobs) *SAJobExe {
	jb := &SAJobExe{jobs: jobs, st_time: OsTime()}

	jb.job_id = job_id
	jb.app = app
	jb.node = node
	jb.dirPath = dirPath
	jb.programName = programName
	jb.input = input

	return jb
}

func (jb *SAJobExe) Run() {
	defer jb.done.Store(true)

	cmd := exec.Command("."+jb.dirPath+jb.programName, strconv.Itoa(jb.jobs.base.services.port), jb.job_id)
	//cmd.Dir = jb.dirPath

	cmd_out, err := cmd.CombinedOutput()
	if err != nil {
		jb.outErr = errors.New(err.Error() + ": " + string(cmd_out))
	}

	jb.outCmd = cmd_out
	jb.dt_time = OsTime() - jb.st_time
}

func (jb *SAJobExe) GetProgress() (string, float64) {
	dt := OsTime() - jb.st_time
	return fmt.Sprintf("Executing %s", jb.programName), dt / jb.jobs.exe_stats.time_avg
}

func (jb *SAJobExe) RenderProgress(y *int) bool {
	ui := jb.jobs.base.ui

	str, proc := jb.GetProgress()
	ui.Comp_text(0, *y, 1, 1, fmt.Sprintf("%s ... %.1f%%", str, proc*100), 0)
	(*y)++

	return true
}

func (jb *SAJobExe) PostRun() {

	node := jb.node.FindPath(jb.app.root)
	if node == nil {
		fmt.Printf("Warning: SAJobExe node '%s' not found\n", jb.node.String())
		return
	}

	node.Code.cmd_output = string(jb.outCmd)
	if jb.outErr == nil {
		node.Code.SetOutput(jb.outJs)
	} else {
		node.Code.exe_err = jb.outErr
	}

	fmt.Printf("SAJobExe '%s' finished in %f\n", jb.programName, jb.dt_time)
}

type SAJobWhisperCpp struct {
	jobs    *SAJobs
	st_time float64

	app   *SAApp
	node  SANodePath
	model string
	blob  OsBlob
	props *SAServiceWhisperCppProps

	output []byte
	outErr error

	dt_time float64
	done    atomic.Bool
}

func NewSAJobWhisperCpp(app *SAApp, node SANodePath, model string, blob OsBlob, props *SAServiceWhisperCppProps, jobs *SAJobs) *SAJobWhisperCpp {
	jb := &SAJobWhisperCpp{jobs: jobs, st_time: OsTime()}

	jb.app = app
	jb.node = node
	jb.model = model
	jb.blob = blob
	jb.props = props

	return jb
}
func (jb *SAJobWhisperCpp) Run() {
	defer jb.done.Store(true)

	wh, err := jb.jobs.getWhisper(jb.model)
	if err == nil {
		jb.output, jb.outErr = wh.Transcribe(jb.model, jb.blob, jb.props)
	} else {
		jb.outErr = err
	}

	jb.dt_time = OsTime() - jb.st_time
}
func (jb *SAJobWhisperCpp) GetProgress() (string, float64) {
	dt := OsTime() - jb.st_time
	return fmt.Sprintf("Whispering %s", jb.model), dt / jb.jobs.compile_stats.time_avg //.........
}
func (jb *SAJobWhisperCpp) RenderProgress(y *int) bool {
	ui := jb.jobs.base.ui

	str, proc := jb.GetProgress()
	ui.Comp_text(0, *y, 1, 1, fmt.Sprintf("%s ... %.1f%%", str, proc*100), 0)
	(*y)++

	return true
}

func (jb *SAJobWhisperCpp) PostRun() {
	fmt.Printf("SAJobWhisperCpp '%s' finished in %f\n", jb.node.String(), jb.dt_time)
}

type SAJobLLamaCpp struct {
	jobs    *SAJobs
	st_time float64

	app   *SAApp
	node  SANodePath
	props *SAServiceLLamaCppProps

	wip_answer string //atomic ...
	stop       bool   //atomic ...

	output []byte
	outErr error

	dt_time float64
	done    atomic.Bool
}

func NewSAJobLLamaCpp(app *SAApp, node SANodePath, props *SAServiceLLamaCppProps, jobs *SAJobs) *SAJobLLamaCpp {
	jb := &SAJobLLamaCpp{jobs: jobs, st_time: OsTime()}

	jb.app = app
	jb.node = node
	jb.props = props

	return jb
}
func (jb *SAJobLLamaCpp) Run() {
	defer jb.done.Store(true)

	wh, err := jb.jobs.getLLama(jb.props.Model)
	if err == nil {
		jb.output, jb.outErr = wh.Complete(jb.props, &jb.wip_answer, &jb.stop)
	} else {
		jb.outErr = err
	}

	jb.dt_time = OsTime() - jb.st_time
}
func (jb *SAJobLLamaCpp) GetProgress() (string, float64) {
	dt := OsTime() - jb.st_time
	return fmt.Sprintf("llama is completing"), dt / jb.jobs.compile_stats.time_avg //.........
}
func (jb *SAJobLLamaCpp) RenderProgress(y *int) bool {
	ui := jb.jobs.base.ui

	str, proc := jb.GetProgress()
	ui.Comp_text(0, *y, 1, 1, fmt.Sprintf("%s ... %.1f%%", str, proc*100), 0)
	(*y)++

	ui.Comp_textSelectMulti(0, *y, 1, 5, jb.wip_answer, OsV2{0, 0}, true, true, false)
	*y = *y + 5

	if ui.Comp_button(0, *y, 1, 1, "Stop", Comp_buttonProp().SetError(true)) > 0 {
		jb.stop = true
	}
	(*y)++

	return true
}

func (jb *SAJobLLamaCpp) PostRun() {
	fmt.Printf("SAJobLLamaCpp '%s' finished in %f\n", jb.node.String(), jb.dt_time)
}

type SAJobOpenAI struct {
	jobs    *SAJobs
	st_time float64

	app   *SAApp
	node  SANodePath
	props *SAServiceOpenAIProps

	wip_answer string //atomic ...
	stop       bool   //atomic ...

	output []byte
	outErr error

	dt_time float64
	done    atomic.Bool
}

func NewSAJobOpenAI(app *SAApp, node SANodePath, props *SAServiceOpenAIProps, jobs *SAJobs) *SAJobOpenAI {
	jb := &SAJobOpenAI{jobs: jobs, st_time: OsTime()}

	jb.app = app
	jb.node = node
	jb.props = props

	return jb
}
func (jb *SAJobOpenAI) Run() {
	defer jb.done.Store(true)

	wh, err := jb.jobs.getOpenAI()
	if err == nil {
		jb.output, jb.outErr = wh.Complete(jb.props, &jb.wip_answer, &jb.stop)
	} else {
		jb.outErr = err
	}

	jb.dt_time = OsTime() - jb.st_time
}
func (jb *SAJobOpenAI) GetProgress() (string, float64) {
	dt := OsTime() - jb.st_time
	return fmt.Sprintf("openAI is completing"), dt / jb.jobs.compile_stats.time_avg //.........
}
func (jb *SAJobOpenAI) RenderProgress(y *int) bool {
	ui := jb.jobs.base.ui

	str, proc := jb.GetProgress()
	ui.Comp_text(0, *y, 1, 1, fmt.Sprintf("%s ... %.1f%%", str, proc*100), 0)
	(*y)++

	ui.Comp_textSelectMulti(0, *y, 1, 5, jb.wip_answer, OsV2{0, 0}, true, true, false)
	*y = *y + 5

	if ui.Comp_button(0, *y, 1, 1, "Stop", Comp_buttonProp().SetError(true)) > 0 {
		jb.stop = true
	}
	(*y)++

	return true
}
func (jb *SAJobOpenAI) PostRun() {
	fmt.Printf("SAJobOpenAI '%s' finished in %f\n", jb.node.String(), jb.dt_time)
}

type SAJobNet struct {
	jobs    *SAJobs
	st_time float64

	app  *SAApp
	node SANodePath
	path string
	url  string

	stop            atomic.Bool
	stop_and_delete atomic.Bool

	recv_bytes  int64
	final_bytes int64

	stat_time float64
	stat_recv atomic.Uint64

	//output []byte
	outErr error

	dt_time float64
	done    atomic.Bool
}

func NewSAJobNet(app *SAApp, node SANodePath, path string, url string, jobs *SAJobs) *SAJobNet {
	jb := &SAJobNet{jobs: jobs, st_time: OsTime()}

	jb.app = app
	jb.node = node
	jb.path = path
	jb.url = url

	return jb
}

var g_SAServiceNet_flagTimeout = flag.Duration("timeout", 30*time.Minute, "HTTP timeout")

func (jb *SAJobNet) Run() {
	defer jb.done.Store(true)

	path := jb.path + ".temp"

	//prepare temp file
	flag := os.O_CREATE | os.O_WRONLY
	if OsFileExists(path) {
		flag = os.O_APPEND | os.O_WRONLY
	}
	file, err := os.OpenFile(path, flag, 0666)
	if err != nil {
		jb.outErr = err
		return
	}

	//prepare client
	req, err := http.NewRequest("GET", jb.url, nil)
	if err != nil {
		file.Close()
		jb.outErr = err
		return
	}
	//req.Header.Set("User-Agent", "skyalt")

	//resume download
	file_bytes, err := file.Seek(0, io.SeekEnd)
	if err != nil {
		file.Close()
		jb.outErr = err
		return
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
		jb.outErr = err
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		file.Close()
		jb.outErr = errors.New(resp.Status)
		return
	}
	jb.recv_bytes = file_bytes
	jb.final_bytes = file_bytes + resp.ContentLength

	// Loop
	data := make([]byte, 1024*64)
	for jb.jobs.base.services.online && !jb.stop.Load() && !jb.stop_and_delete.Load() {

		//download
		n, err := resp.Body.Read(data)
		if err != nil {
			jb.outErr = err
			break
		}
		//save
		m, err := file.Write(data[:n])
		if err != nil {
			jb.outErr = err
			break
		}

		jb.recv_bytes += int64(m)

		jb.stat_recv.Add(uint64(m))
	}

	file.Close()

	if jb.recv_bytes == jb.final_bytes {
		OsFileRename(path, jb.path) //<name>.temp -> <name>
	} else {
		if jb.stop_and_delete.Load() {
			OsFileRemove(path)
		}
	}

	if jb.stop.Load() || jb.stop_and_delete.Load() {
		jb.outErr = fmt.Errorf("downloading canceled")
	}

	jb.dt_time = OsTime() - jb.st_time
}
func (jb *SAJobNet) getProcDone() float64 {
	if jb.final_bytes > 0 {
		return float64(jb.recv_bytes) / float64(jb.final_bytes)
	}
	return 0
}
func (jb *SAJobNet) getAvgRecvBytesPerSec() float64 {
	act_time := OsTime()

	old_time := jb.stat_time
	bytes := jb.stat_recv.Load()

	if (act_time - jb.stat_time) > 3 {
		//reset
		jb.stat_time = act_time
		bytes = jb.stat_recv.Swap(0)
	}

	return float64(bytes) / (act_time - old_time)
}
func (jb *SAJobNet) GetProgress() (string, float64) {
	speed := jb.getAvgRecvBytesPerSec()

	remain_sec := 0
	if speed > 0 {
		remain_sec = int(float64(jb.final_bytes-jb.recv_bytes) / speed)
	}

	now := time.Now()
	predict := now.Add(time.Duration(remain_sec) * time.Second)
	diff := predict.Sub(now)

	return fmt.Sprintf("Downloading '%.10s': %.1f/%.1fMB, %.3fMB/s %v", jb.url, float64(jb.recv_bytes)/(1024*1024), float64(jb.final_bytes)/(1024*1024), speed/(1024*1024), diff), jb.getProcDone()
}
func (jb *SAJobNet) RenderProgress(y *int) bool {
	ui := jb.jobs.base.ui

	str, proc := jb.GetProgress()
	ui.Comp_text(0, *y, 1, 1, fmt.Sprintf("%s ... %.1f%%", str, proc*100), 0)
	(*y)++

	ok := true
	ui.Div_start(0, *y, 1, 1)
	{
		ui.Div_colMax(0, 100)
		ui.Div_colMax(1, 100)
		if ui.Comp_button(0, 0, 1, 1, "Stop", Comp_buttonProp().SetError(true)) > 0 {
			jb.stop.Store(true)
			ok = false
		}
		if ui.Comp_button(1, 0, 1, 1, "Stop & Delete file", Comp_buttonProp().SetError(true)) > 0 {
			jb.stop_and_delete.Store(true)
			ok = false
		}
	}
	ui.Div_end()
	(*y)++
	return ok
}
func (jb *SAJobNet) PostRun() {
	fmt.Printf("SAJobNet '%s' finished in %f\n", jb.url, jb.dt_time)
}

type SAJobs struct {
	base *SABase

	compile_stats SAJobTimeStat
	exe_stats     SAJobTimeStat

	compiles []*SAJobCompile
	exes     []*SAJobExe
	whispers []*SAJobWhisperCpp
	llamas   []*SAJobLLamaCpp
	oais     []*SAJobOpenAI
	nets     []*SAJobNet

	whisperCpp *SAServiceWhisperCpp
	llamaCpp   *SAServiceLLamaCpp
	oai        *SAServiceOpenAI
	//net        *SAServiceNet

	lock sync.Mutex

	last_job_id int
}

func NewSAJobs(base *SABase) *SAJobs {
	jobs := &SAJobs{base: base}
	jobs.compile_stats = InitSAJobStats(1)
	jobs.exe_stats = InitSAJobStats(1)

	jobs.last_job_id = int(rand.Int31())
	return jobs
}

func (jobs *SAJobs) Destroy() {

	//close all the jobs ...........

	if jobs.whisperCpp != nil {
		jobs.whisperCpp.Destroy()
	}
	if jobs.llamaCpp != nil {
		jobs.llamaCpp.Destroy()
	}
	if jobs.oai != nil {
		jobs.oai.Destroy()
	}
	/*if jobs.net != nil {
		jobs.net.Destroy()
	}*/
}

func (jobs *SAJobs) AddCompile(app *SAApp, node SANodePath, dirPath string, fileName string) *SAJobCompile {
	jobs.lock.Lock()
	defer jobs.lock.Unlock()

	jb := NewSAJobCompile(app, node, dirPath, fileName, jobs)
	jobs.compiles = append(jobs.compiles, jb)
	go jb.Run()
	return jb
}
func (jobs *SAJobs) AddExe(app *SAApp, node SANodePath, dirPath string, programName string, input []byte) *SAJobExe {
	jobs.lock.Lock()
	defer jobs.lock.Unlock()

	jobs.last_job_id++
	jb := NewSAJobExe(strconv.Itoa(jobs.last_job_id), app, node, dirPath, programName, input, jobs)
	jobs.exes = append(jobs.exes, jb)
	go jb.Run()
	return jb
}
func (jobs *SAJobs) AddWhisper(app *SAApp, node SANodePath, model string, blob OsBlob, props *SAServiceWhisperCppProps) *SAJobWhisperCpp {
	jobs.lock.Lock()
	defer jobs.lock.Unlock()

	jb := NewSAJobWhisperCpp(app, node, model, blob, props, jobs)
	jobs.whispers = append(jobs.whispers, jb)
	go jb.Run()
	return jb
}
func (jobs *SAJobs) AddLLama(app *SAApp, node SANodePath, props *SAServiceLLamaCppProps) *SAJobLLamaCpp {
	jobs.lock.Lock()
	defer jobs.lock.Unlock()

	jb := NewSAJobLLamaCpp(app, node, props, jobs)
	jobs.llamas = append(jobs.llamas, jb)
	go jb.Run()
	return jb
}
func (jobs *SAJobs) AddOpenAI(app *SAApp, node SANodePath, props *SAServiceOpenAIProps) *SAJobOpenAI {
	jobs.lock.Lock()
	defer jobs.lock.Unlock()

	jb := NewSAJobOpenAI(app, node, props, jobs)
	jobs.oais = append(jobs.oais, jb)
	go jb.Run()
	return jb
}

func (jobs *SAJobs) AddNet(app *SAApp, node SANodePath, path string, url string) *SAJobNet {
	jobs.lock.Lock()
	defer jobs.lock.Unlock()

	jb := NewSAJobNet(app, node, path, url, jobs)
	jobs.nets = append(jobs.nets, jb)
	go jb.Run()
	return jb
}

func (jobs *SAJobs) getWhisper(init_model string) (*SAServiceWhisperCpp, error) {
	jobs.lock.Lock()
	defer jobs.lock.Unlock()

	var err error
	if jobs.whisperCpp == nil {
		jobs.whisperCpp, err = NewSAServiceWhisperCpp(jobs, "http://127.0.0.1", "8090", init_model)
	}
	return jobs.whisperCpp, err
}

func (jobs *SAJobs) getLLama(init_model string) (*SAServiceLLamaCpp, error) {
	jobs.lock.Lock()
	defer jobs.lock.Unlock()

	var err error
	if jobs.llamaCpp == nil {
		jobs.llamaCpp, err = NewSAServiceLLamaCpp(jobs, "http://127.0.0.1", "8091", init_model)
	}
	return jobs.llamaCpp, err
}

func (jobs *SAJobs) getOpenAI() (*SAServiceOpenAI, error) {
	jobs.lock.Lock()
	defer jobs.lock.Unlock()

	if !jobs.base.services.online {
		return nil, fmt.Errorf("internet is disabled(Menu:Settings:Internet Connection)")
	}

	if jobs.oai == nil {
		jobs.oai = NewSAServiceOpenAI(jobs)
	}
	return jobs.oai, nil
}

/*func (jobs *SAJobs) GetNet() (*SAServiceNet, error) {
	if !jobs.base.services.online {
		return nil, fmt.Errorf("internet is disabled(Menu:Settings:Internet Connection)")
	}

	if jobs.net == nil {
		jobs.net = NewSAServiceNet(jobs)
	}
	return jobs.net, nil
}*/

func (jobs *SAJobs) FindAppProgress(app *SAApp) (string, float64) {
	jobs.lock.Lock()
	defer jobs.lock.Unlock()

	for _, jb := range jobs.whispers {
		if jb.app == app {
			return jb.GetProgress()
		}
	}
	for _, jb := range jobs.llamas {
		if jb.app == app {
			return jb.GetProgress()
		}
	}
	for _, jb := range jobs.oais {
		if jb.app == app {
			return jb.GetProgress()
		}
	}
	for _, jb := range jobs.nets {
		if jb.app == app {
			return jb.GetProgress()
		}
	}

	for _, jb := range jobs.exes {
		if jb.app == app {
			return jb.GetProgress()
		}
	}

	for _, jb := range jobs.compiles {
		if jb.app == app {
			return jb.GetProgress()
		}
	}

	return "", -1
}

func (jobs *SAJobs) RenderAppProgress(app *SAApp) bool {
	jobs.lock.Lock()
	defer jobs.lock.Unlock()

	y := 0

	ui := jobs.base.ui
	ui.Div_colMax(0, 20)

	ok := false
	for _, jb := range jobs.compiles {
		if jb.app == app {
			if jb.RenderProgress(&y) {
				ok = true
			}
		}
	}
	for _, jb := range jobs.exes {
		if jb.app == app {
			if jb.RenderProgress(&y) {
				ok = true
			}
		}
	}
	for _, jb := range jobs.whispers {
		if jb.app == app {
			if jb.RenderProgress(&y) {
				ok = true
			}
		}
	}
	for _, jb := range jobs.llamas {
		if jb.app == app {
			if jb.RenderProgress(&y) {
				ok = true
			}
		}
	}
	for _, jb := range jobs.oais {
		if jb.app == app {
			if jb.RenderProgress(&y) {
				ok = true
			}
		}
	}
	for _, jb := range jobs.nets {
		if jb.app == app {
			if jb.RenderProgress(&y) {
				ok = true
			}
		}
	}
	return ok
}

func (jobs *SAJobs) FindJobExe(job_id string) *SAJobExe {
	jobs.lock.Lock()
	defer jobs.lock.Unlock()

	for _, jb := range jobs.exes {
		if jb.job_id == job_id {
			return jb
		}
	}
	return nil
}

func (jobs *SAJobs) Tick() {
	jobs.lock.Lock()
	defer jobs.lock.Unlock()

	for i := len(jobs.compiles) - 1; i >= 0; i-- {
		jb := jobs.compiles[i]
		if jb.done.Load() {
			jobs.compile_stats.Add(jb.dt_time)
			jb.PostRun()
			jobs.compiles = append(jobs.compiles[:i], jobs.compiles[i+1:]...) //remove
		}
	}

	for i := len(jobs.exes) - 1; i >= 0; i-- {
		jb := jobs.exes[i]
		if jb.done.Load() {
			jobs.exe_stats.Add(jb.dt_time)
			jb.PostRun()
			jobs.exes = append(jobs.exes[:i], jobs.exes[i+1:]...) //remove
		}
	}

	for i := len(jobs.whispers) - 1; i >= 0; i-- {
		jb := jobs.whispers[i]
		if jb.done.Load() {
			jb.PostRun()
			jobs.whispers = append(jobs.whispers[:i], jobs.whispers[i+1:]...) //remove
		}
	}

	for i := len(jobs.llamas) - 1; i >= 0; i-- {
		jb := jobs.llamas[i]
		if jb.done.Load() {
			jb.PostRun()
			jobs.llamas = append(jobs.llamas[:i], jobs.llamas[i+1:]...) //remove
		}
	}

	for i := len(jobs.oais) - 1; i >= 0; i-- {
		jb := jobs.oais[i]
		if jb.done.Load() {
			jb.PostRun()
			jobs.oais = append(jobs.oais[:i], jobs.oais[i+1:]...) //remove
		}
	}

	for i := len(jobs.nets) - 1; i >= 0; i-- {
		jb := jobs.nets[i]
		if jb.done.Load() {
			jb.PostRun()
			jobs.nets = append(jobs.nets[:i], jobs.nets[i+1:]...) //remove
		}
	}
}
