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
	"strings"
)

func SAExe_Code_python(node *SANode) bool {

	codeAttr := node.GetAttr("code", "")

	//check code
	code := codeAttr.GetString()
	if code == "" {
		codeAttr.SetErrorExe("empty")
		return false
	}
	if strings.Contains(strings.ToLower(code), "import") {
		codeAttr.SetErrorExe("Code contains 'import' keyword")
		return false
	}

	//set attributes to json
	attrsList := make(map[string]interface{})
	for _, a := range node.Attrs {
		if a.Name == "code" || a.IsOutput() {
			continue //skip
		}
		attrsList[a.Name] = a.GetResult().value
	}

	//run python on service server
	outAttrs, errStr, err := node.app.base.service_python.Exec(code, attrsList)
	if err != nil {
		codeAttr.SetErrorExe(err.Error())
		return false
	}
	if errStr != "" {
		codeAttr.SetErrorExe(errStr)
		return false
	}

	//set values into output attributes
	for name, value := range outAttrs {
		if !strings.HasPrefix(name, "_") {
			continue //skip
		}

		attr := node.GetAttr(name, "")
		switch vv := value.(type) {
		case string:
			attr.GetResult().SetString(vv)
		case float64:
			attr.GetResult().SetNumber(vv)
		case int:
			attr.GetResult().SetNumber(float64(vv))
		default:
			fmt.Println("Unsupported format")
		}
	}

	//remove un-used(attr.exeMark==false) outputs attributes
	for i := len(node.Attrs) - 1; i >= 0; i-- {
		if node.Attrs[i].IsOutput() {
			if !node.Attrs[i].exeMark {
				node.Attrs = append(node.Attrs[:i], node.Attrs[i+1:]...) //remove
			}
		}
	}

	return true
}
