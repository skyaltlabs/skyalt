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

func SAExe_NN_whisper(node *SANode) bool {
	modelAttr := node.GetAttr("model", "\"models/ggml-tiny.en.bin\"")
	audioAttr := node.GetAttr("audio", "") //blob
	textAttr := node.GetAttr("_text", "")
	doneAttr := node.GetAttr("_done", "0")

	if modelAttr.GetString() == "" {
		modelAttr.SetErrorExe("empty")
		return false
	}

	str, progress, done, err := node.app.base.service_whisper.Translate(modelAttr.GetString(), audioAttr.GetBlob())
	if err != nil {
		node.SetError(err.Error())
		return false
	}

	node.progress = progress

	doneAttr.SetOutBlob([]byte(OsTrnString(done, "1", "0")))
	textAttr.SetOutBlob([]byte(str))

	return true
}
