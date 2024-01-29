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

type UiPaintTextHistoryItem struct {
	str string
	cur int
}

type UiPaintTextHistory struct {
	uid *UiLayoutDiv

	items []UiPaintTextHistoryItem

	act          int
	lastAddTicks int64
}

func NewUiPaintTextHistoryItem(uid *UiLayoutDiv, init UiPaintTextHistoryItem) *UiPaintTextHistory {
	var his UiPaintTextHistory
	his.uid = uid

	his.items = append(his.items, init)
	his.lastAddTicks = OsTicks()

	return &his
}

func (his *UiPaintTextHistory) Add(value UiPaintTextHistoryItem) bool {

	//same as previous
	if his.items[his.act].str == value.str {
		return false
	}

	//cut all after
	his.items = his.items[:his.act+1]

	//adds new snapshot
	his.items = append(his.items, value)
	his.act++
	his.lastAddTicks = OsTicks()

	return true
}
func (his *UiPaintTextHistory) AddWithTimeOut(value UiPaintTextHistoryItem) bool {
	if !OsIsTicksIn(his.lastAddTicks, 500) {
		return his.Add(value)
	}
	return false
}

func (his *UiPaintTextHistory) Backward(init UiPaintTextHistoryItem) UiPaintTextHistoryItem {

	his.Add(init)

	if his.act-1 >= 0 {
		his.act--
	}
	return his.items[his.act]
}
func (his *UiPaintTextHistory) Forward() UiPaintTextHistoryItem {
	if his.act+1 < len(his.items) {
		his.act++
	}
	return his.items[his.act]
}

type UiPaintTextHistoryArray struct {
	items []*UiPaintTextHistory
}

func (his *UiPaintTextHistoryArray) FindOrAdd(uid *UiLayoutDiv, init UiPaintTextHistoryItem) *UiPaintTextHistory {

	//finds
	for _, it := range his.items {
		if it.uid == uid {
			return it
		}
	}

	//adds
	it := NewUiPaintTextHistoryItem(uid, init)
	his.items = append(his.items, it)
	return it
}
