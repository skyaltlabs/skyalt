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

type Rec struct {
	value SAValue
}

func NewRec() *Rec {
	var rec Rec
	rec.value = InitSAValue()
	return &rec
}

func (rec *Rec) Is() bool {
	if rec == nil {
		return false
	}
	return rec.value.Is()
}

func (A *Rec) Cmp(B *Rec) int {
	if A != nil && B != nil {
		return A.value.Cmp(&B.value)
	}

	return 1 // different
}
