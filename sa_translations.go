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
	"fmt"
	"os"
	"strings"
)

func SATranslations_fromJson(js []byte, langs []string) ([]byte, error) {

	keys := make(map[string]string)

	err := json.Unmarshal(js, &keys)
	if err != nil {
		fmt.Printf("Unmarshal() failed: %v\n", err)
		return nil, err
	}

	//from 'id.lang=value' => 'id=value' based on 'langs' priority
	ids := make(map[string]string)
	for _, act_lang := range langs {
		for key, value := range keys {

			d := strings.IndexByte(key, '.')
			if d >= 0 {
				name := key[:d]
				lang := key[d+1:]

				if lang == act_lang {
					_, found := ids[name]
					if !found {
						ids[name] = value
					}
				}

			}
		}
	}

	//map => json string
	js, err = json.MarshalIndent(ids, "", "")
	if err != nil {
		return nil, fmt.Errorf("MarshalIndent() failed: %w", err)
	}
	return js, nil
}

func SATranslations_fromJsonFile(path string, langs []string) ([]byte, error) {
	js, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("ReadFile(%s) failed: %w", path, err)
	}

	return SATranslations_fromJson(js, langs)
}