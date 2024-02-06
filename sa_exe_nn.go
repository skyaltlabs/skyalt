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

func SAExe_NN_whisper_cpp(node *SANode) bool {
	modelAttr := node.GetAttr("model", "\"models/ggml-tiny.en.bin\"")
	audioAttr := node.GetAttr("audio", "") //blob
	_textAttr := node.GetAttr("_text", "")

	if modelAttr.GetString() == "" {
		modelAttr.SetErrorExe("empty")
		return false
	}

	str, progress, _, err := node.app.base.service_whisper_cpp.Translate(modelAttr.GetString(), audioAttr.GetBlob())
	if err != nil {
		node.SetError(err.Error())
		return false
	}

	node.progress = progress
	_textAttr.SetOutBlob([]byte(str))

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

		triggerAttr.exePostExpSet = "0"
	}

	return true
}
