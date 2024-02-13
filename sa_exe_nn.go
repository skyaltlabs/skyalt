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
	"encoding/json"
	"net/url"
	"path/filepath"
	"strings"
)

var g_whisper_modelList = []string{"", "ggml-tiny.en", "ggml-tiny", "ggml-base.en", "ggml-base", "ggml-small.en", "ggml-small", "ggml-medium.en", "ggml-medium", "ggml-large-v1", "ggml-large-v2", "ggml-large-v3"}

func SAExe_NN_whisper_cpp(node *SANode) bool {

	labels := ";" //empty
	modelsFolder := "services/whisper.cpp/models/"
	//labels
	for _, m := range g_whisper_modelList {
		if m != "" { //1st is empty
			if OsFileExists(filepath.Join(modelsFolder, m+".bin")) {
				labels += m + ";"
			}
		}
	}
	labels, _ = strings.CutSuffix(labels, ";")

	modelAttr := node.GetAttrUi("model", "", SAAttrUi_COMBO(labels, labels))
	modelAttr.Ui = SAAttrUi_COMBO(labels, labels) //rewrite actual value as well(not only defaultUi)

	modelPath := filepath.Join("models", modelAttr.GetString()+".bin")

	audioAttr := node.GetAttr("input", "") //blob

	var props SAServiceWhisperCppProps
	{
		//Documentation: https://github.com/ggerganov/whisper.cpp/tree/master/examples/server

		props.Offset_t = node.GetAttrUi("offset_t", 0, SAAttrUiValue{}).GetInt()
		props.Offset_n = node.GetAttrUi("offset_n", 0, SAAttrUiValue{}).GetInt()
		props.Duration = node.GetAttrUi("duration", 0, SAAttrUiValue{}).GetInt()
		props.Max_context = node.GetAttrUi("max_context", -1, SAAttrUiValue{}).GetInt()
		props.Max_len = node.GetAttrUi("max_len", 0, SAAttrUiValue{}).GetInt()
		props.Best_of = node.GetAttrUi("best_of", 2, SAAttrUiValue{}).GetInt()
		props.Beam_size = node.GetAttrUi("beam_size", -1, SAAttrUiValue{}).GetInt()

		props.Word_thold = node.GetAttrUi("word_thold", 0.01, SAAttrUiValue{}).GetFloat()
		props.Entropy_thold = node.GetAttrUi("entropy_thold", 2.4, SAAttrUiValue{}).GetFloat()
		props.Logprob_thold = node.GetAttrUi("logprob_thold", -1, SAAttrUiValue{}).GetFloat()

		props.Translate = node.GetAttrUi("translate", 0, SAAttrUi_SWITCH).GetBool()
		props.Diarize = node.GetAttrUi("diarize", 0, SAAttrUi_SWITCH).GetBool()
		props.Tinydiarize = node.GetAttrUi("tinydiarize", 0, SAAttrUi_SWITCH).GetBool()
		props.Split_on_word = node.GetAttrUi("split_on_word", 0, SAAttrUi_SWITCH).GetBool()
		props.No_timestamps = node.GetAttrUi("no_timestamps", 0, SAAttrUi_SWITCH).GetBool()

		props.Language = node.GetAttrUi("language", "", SAAttrUiValue{}).GetString()
		props.Detect_language = node.GetAttrUi("detect_language", 0, SAAttrUi_SWITCH).GetBool()

		props.Temperature = node.GetAttrUi("temperature", 0, SAAttrUiValue{}).GetFloat()
		props.Temperature_inc = node.GetAttrUi("temperature_inc", 0.2, SAAttrUiValue{}).GetFloat()

		props.Response_format = node.GetAttrUi("response_format", "verbose_json", SAAttrUi_COMBO("verbose_json;json;text;srt;vtt", "verbose_json;json;text;srt;vtt")).GetString()
	}

	_outAttr := node.GetAttr("_out", "")

	if modelAttr.GetString() == "" {
		modelAttr.SetErrorStr("empty")
		return false
	}

	propHash, err := props.Hash()
	if err != nil {
		modelAttr.SetError(err)
		return false
	}

	//try find in cache
	str, found := node.app.base.service_whisper_cpp.FindCache(modelPath, audioAttr.GetBlob(), propHash)
	if found {
		_outAttr.SetOutBlob([]byte(str))
	} else {
		//add job
		job := node.app.jobs.AddJob(node)
		go SAJob_NN_whisper_cpp(job, modelPath, audioAttr.GetBlob(), &props)
	}
	return true
}

func SAExe_NN_whisper_cpp_downloader(node *SANode) bool {
	serverAttr := node.GetAttr("server", "https://huggingface.co/ggerganov/whisper.cpp/resolve/main")
	if serverAttr.GetString() == "" {
		serverAttr.SetErrorStr("empty")
		return false
	}
	folderAttr := node.GetAttr("folder", "services/whisper.cpp/models/")
	if folderAttr.GetString() == "" {
		folderAttr.SetErrorStr("empty")
		return false
	}

	//labels
	var labels string
	for _, m := range g_whisper_modelList {
		if m != "" { //1st is empty
			labels += m
			if OsFileExists(filepath.Join(folderAttr.GetString(), m+".bin")) {
				labels += "(found)"
			}
		}
		labels += ";"
	}
	labels, _ = strings.CutSuffix(labels, ";")

	//pick model
	modelAttr := node.GetAttrUi("model", "", SAAttrUi_COMBO(labels, ""))
	modelAttr.Ui = SAAttrUi_COMBO(labels, "") //rewrite actual value as well(not only defaultUi)
	id := modelAttr.GetInt()
	if id > 0 {
		urll := ""
		{
			u, err := url.Parse(serverAttr.GetString())
			if err != nil {
				serverAttr.SetError(err)
				return false
			}
			u.Path = filepath.Join(u.Path, g_whisper_modelList[id]+".bin")
			urll = u.String()
		}

		dst := filepath.Join(folderAttr.GetString(), g_whisper_modelList[id]+".bin")

		//add job
		job := node.app.jobs.AddJob(node)
		go SAJob_downloader(job, urll, dst, g_whisper_modelList[id])

		//reset in next tick
		modelAttr.AddSetAttr("0")
	}

	return true
}

type SAExe_llama_cpp_model struct {
	url_base string
	name     string
}

var g_llama_modelList = []SAExe_llama_cpp_model{
	{"", ""},

	{"https://huggingface.co/TheBloke/Llama-2-7b-Chat-GGUF/resolve/main", "llama-2-7b-chat.Q4_K_S.gguf"},
	{"https://huggingface.co/TheBloke/Llama-2-7b-Chat-GGUF/resolve/main", "llama-2-7b-chat.Q6_K.gguf"},

	{"https://huggingface.co/TheBloke/phi-2-GGUF/resolve/main", "phi-2.Q4_K_S.gguf"},
	{"https://huggingface.co/TheBloke/phi-2-GGUF/resolve/main", "phi-2.Q6_K.gguf"},

	{"https://huggingface.co/TheBloke/Mistral-7B-Instruct-v0.1-GGUF/resolve/main", "mistral-7b-instruct-v0.1.Q4_K_S.gguf"},
	{"https://huggingface.co/TheBloke/Mistral-7B-Instruct-v0.1-GGUF/resolve/main", "mistral-7b-instruct-v0.1.Q6_K.gguf"},

	{"https://huggingface.co/TheBloke/CodeLlama-7B-Instruct-GGUF/resolve/main", "codellama-7b-instruct.Q4_K_S.gguf"},
	{"https://huggingface.co/TheBloke/CodeLlama-7B-Instruct-GGUF/resolve/main", "codellama-7b-instruct.Q6_K.gguf"},
}

func SAExe_NN_llama_cpp_downloader(node *SANode) bool {
	folderAttr := node.GetAttr("folder", "services/llama.cpp/models/")
	if folderAttr.GetString() == "" {
		folderAttr.SetErrorStr("empty")
		return false
	}

	//labels
	var labels string
	for _, m := range g_llama_modelList {
		if m.name != "" { //1st is empty
			labels += m.name
			if OsFileExists(filepath.Join(folderAttr.GetString(), m.name)) {
				labels += "(found)"
			}
		}
		labels += ";"
	}
	labels, _ = strings.CutSuffix(labels, ";")

	//pick model
	modelAttr := node.GetAttrUi("model", "", SAAttrUi_COMBO(labels, ""))
	modelAttr.Ui = SAAttrUi_COMBO(labels, "") //rewrite actual value as well(not only defaultUi)
	id := modelAttr.GetInt()
	if id > 0 {
		urll := ""
		{
			u, err := url.Parse(g_llama_modelList[id].url_base)
			if err != nil {
				node.SetError(err)
				return false
			}
			u.Path = filepath.Join(u.Path, g_llama_modelList[id].name)
			urll = u.String()
		}

		dst := filepath.Join(folderAttr.GetString(), g_llama_modelList[id].name)

		//add job
		job := node.app.jobs.AddJob(node)
		go SAJob_downloader(job, urll, dst, g_llama_modelList[id].name)

		//reset in next tick
		modelAttr.AddSetAttr("0")
	}

	return true
}

func SAExe_NN_llama_cpp(node *SANode) bool {

	modelsFolder := "services/llama.cpp/models/"
	labels := ";" //empty
	modelFiles := OsFileListBuild(modelsFolder, "", true)
	for _, m := range modelFiles.Subs {
		if !m.IsDir && !strings.HasPrefix(m.Name, "ggml-vocab") {
			labels += m.Name + ";"
		}
	}
	labels, _ = strings.CutSuffix(labels, ";")

	modelAttr := node.GetAttrUi("model", "", SAAttrUi_COMBO(labels, labels))
	modelAttr.Ui = SAAttrUi_COMBO(labels, labels) //rewrite actual value as well(not only defaultUi)
	modelPath := filepath.Join("models", modelAttr.GetString())

	var props SAServiceLLamaCppProps
	{
		//Documentation: https://github.com/ggerganov/llama.cpp/tree/master/examples/server

		props.Prompt = node.GetAttrUi("prompt", "This is a conversation between User and Llama, a friendly chatbot. Llama is helpful, kind, honest, good at writing, and never fails to answer any requests immediately and with precision.\n\nUser: How Are you doing?\nLlama:", SAAttrUi_CODE).GetString()

		stopAttr := node.GetAttr("stop", []byte(`["</s>", "Llama:", "User:"]`))
		err := json.Unmarshal(stopAttr.GetBlob().data, &props.Stop)
		if err != nil {
			stopAttr.SetError(err)
		}

		props.Seed = node.GetAttr("seed", -1).GetInt()
		props.N_predict = node.GetAttr("n_predict", 400).GetInt()
		props.Temperature = node.GetAttr("temperature", 0.8).GetFloat()
		props.Dynatemp_range = node.GetAttr("dynatemp_range", 0.0).GetFloat()
		props.Dynatemp_exponent = node.GetAttr("dynatemp_exponent", 1.0).GetFloat()
		props.Repeat_last_n = node.GetAttr("repeat_last_n", 256).GetInt()
		props.Repeat_penalty = node.GetAttr("repeat_penalty", 1.18).GetFloat()
		props.Top_k = node.GetAttr("top_k", 40).GetInt()
		props.Top_p = node.GetAttr("top_p", 0.5).GetFloat()
		props.Min_p = node.GetAttr("min_p", 0.05).GetFloat()
		props.Tfs_z = node.GetAttr("tfs_z", 1.0).GetFloat()
		props.Typical_p = node.GetAttr("typical_p", 1.0).GetFloat()
		props.Presence_penalty = node.GetAttr("presence_penalty", 0.0).GetFloat()
		props.Frequency_penalty = node.GetAttr("frequency_penalty", 0.0).GetFloat()
		props.Mirostat = node.GetAttr("mirostat", 0).GetInt()
		props.Mirostat_tau = node.GetAttr("mirostat_tau", 5).GetFloat()
		props.Mirostat_eta = node.GetAttr("mirostat_eta", 0.1).GetFloat()
		//Grammar
		props.N_probs = node.GetAttr("n_probs", 0).GetInt()
		//Image_data
		props.Cache_prompt = node.GetAttrUi("cache_prompt", "0", SAAttrUi_SWITCH).GetBool()
		props.Slot_id = node.GetAttr("slot_id", -1).GetInt()
	}

	_outAttr := node.GetAttr("_out", "")

	if modelAttr.GetString() == "" {
		modelAttr.SetErrorStr("empty")
		return false
	}

	propHash, err := props.Hash()
	if err != nil {
		modelAttr.SetError(err)
		return false
	}

	//try find in cache
	str, found := node.app.base.service_llama_cpp.FindCache(modelPath, propHash)
	if found {
		_outAttr.SetOutBlob([]byte(str))
	} else {
		//add job
		job := node.app.jobs.AddJob(node)
		go SAJob_NN_llama_cpp(job, modelPath, &props)
	}
	return true
}
