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
	modelPath := filepath.Join(modelsFolder, modelAttr.GetString()+".bin")

	audioAttr := node.GetAttr("audio", "") //blob

	var props SAServiceWhisperCppProps
	{
		props.Offset_t = node.GetAttrUi("offset_t", "0", SAAttrUiValue{}).GetInt()
		props.Offset_n = node.GetAttrUi("offset_n", "0", SAAttrUiValue{}).GetInt()
		props.Duration = node.GetAttrUi("duration", "0", SAAttrUiValue{}).GetInt()
		props.Max_context = node.GetAttrUi("max_context", "-1", SAAttrUiValue{}).GetInt()
		props.Max_len = node.GetAttrUi("max_len", "0", SAAttrUiValue{}).GetInt()
		props.Best_of = node.GetAttrUi("best_of", "2", SAAttrUiValue{}).GetInt()
		props.Beam_size = node.GetAttrUi("beam_size", "-1", SAAttrUiValue{}).GetInt()

		props.Word_thold = node.GetAttrUi("word_thold", "0.01", SAAttrUiValue{}).GetFloat()
		props.Entropy_thold = node.GetAttrUi("entropy_thold", "2.4", SAAttrUiValue{}).GetFloat()
		props.Logprob_thold = node.GetAttrUi("logprob_thold", "-1", SAAttrUiValue{}).GetFloat()

		props.Translate = node.GetAttrUi("translate", "0", SAAttrUi_SWITCH).GetBool()
		props.Diarize = node.GetAttrUi("diarize", "0", SAAttrUi_SWITCH).GetBool()
		props.Tinydiarize = node.GetAttrUi("tinydiarize", "0", SAAttrUi_SWITCH).GetBool()
		props.Split_on_word = node.GetAttrUi("split_on_word", "0", SAAttrUi_SWITCH).GetBool()
		props.No_timestamps = node.GetAttrUi("no_timestamps", "0", SAAttrUi_SWITCH).GetBool()

		props.Language = node.GetAttrUi("language", "", SAAttrUiValue{}).GetString()
		props.Detect_language = node.GetAttrUi("detect_language", "0", SAAttrUi_SWITCH).GetBool()

		props.Temperature = node.GetAttrUi("temperature", "0", SAAttrUiValue{}).GetFloat()
		props.Temperature_inc = node.GetAttrUi("temperature_inc", "0.2", SAAttrUiValue{}).GetFloat()

		props.Response_format = node.GetAttrUi("response_format", "\"verbose_json\"", SAAttrUi_COMBO("verbose_json;json;text;srt;vtt", "verbose_json;json;text;srt;vtt")).GetString()
	}

	_outAttr := node.GetAttr("_out", "")

	if modelPath == "" {
		modelAttr.SetErrorExe("empty")
		return false
	}

	propHash, err := props.Hash()
	if err != nil {
		modelAttr.SetErrorExe(err.Error())
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
	serverAttr := node.GetAttr("server", "\"https://huggingface.co/ggerganov/whisper.cpp/resolve/main\"")
	if serverAttr.GetString() == "" {
		serverAttr.SetErrorExe("empty")
		return false
	}
	folderAttr := node.GetAttr("folder", "\"services/whisper.cpp/models/\"")
	if folderAttr.GetString() == "" {
		folderAttr.SetErrorExe("empty")
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
				serverAttr.SetErrorExe(err.Error())
				return false
			}
			u.Path = filepath.Join(u.Path, g_whisper_modelList[id]+".bin")
			urll = u.String()
		}

		dst := filepath.Join(folderAttr.GetString(), g_whisper_modelList[id]+".bin")

		//add job
		job := node.app.jobs.AddJob(node)
		go SAJob_NN_whisper_cpp_downloader(job, urll, dst, g_whisper_modelList[id])

		//reset in next tick
		modelAttr.AddSetAttr("0")
	}

	return true
}

func SAExe_NN_llama_cpp(node *SANode) bool {
	triggerAttr := node.GetAttrUi("trigger", "0", SAAttrUi_SWITCH)

	modelAttr := node.GetAttr("model", "\"models/llama-2-7b.Q5_K_M.gguf\"")
	textAttr := node.GetAttr("text", "\"This is a conversation between User and Llama, a friendly chatbot. Llama is helpful, kind, honest, good at writing, and never fails to answer any requests immediately and with precision.\n\nUser: How Are you doing?\nLlama:\"") //blob
	_textAttr := node.GetAttr("_text", "")
	//seed ...

	if modelAttr.GetString() == "" {
		modelAttr.SetErrorExe("empty")
		return false
	}

	if triggerAttr.GetBool() {
		str, progress, _, err := node.app.base.service_llama_cpp.Complete(modelAttr.GetString(), textAttr.GetBlob())
		if err != nil {
			node.SetError(err.Error())
			return false
		}

		node.progress = progress
		_textAttr.SetOutBlob([]byte(str))

		triggerAttr.AddSetAttr("0")
	}

	return true
}
