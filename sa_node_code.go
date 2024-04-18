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
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strconv"
	"strings"

	_ "embed"
)

//go:embed sa_node_const_go.goo
var g_code_const_go string

var g_str_imports = []string{"\"bytes\"", "\"encoding/json\"", "\"fmt\"", "\"io\"", "\"net/http\"", "\"os\"", "\"strconv\""}

type SANodeCodeChat struct {
	User      string
	Assistent string
}
type SANodeCodeImport struct {
	Name string
	Path string
}
type SANodeCodeArg struct {
	Node  string
	Write bool
}

type SANodeCodeFn struct {
	node *SANode

	updated bool
	write   bool
}

type SANodeCodeExePrm struct {
	Node     string
	ListPos  int
	ListNode string
	Attr     string
	Value    interface{}
}
type SANodeCodeExe struct {
	prms []SANodeCodeExePrm
}

type SANodeCode struct {
	node *SANode

	Messages []SANodeCodeChat

	Code string

	func_depends []*SANodeCodeFn

	cmd_output string //terminal

	file_err error
	exe_err  error
	ans_err  error

	exes []SANodeCodeExe

	job_exe *SAJobExe

	job_oai       *SAJobOpenAI //answer is generated
	job_oai_index int
}

func InitSANodeCode(node *SANode) SANodeCode {
	ls := SANodeCode{}
	ls.node = node
	return ls
}

func (ls *SANodeCode) AddExe(exe_prms []SANodeCodeExePrm) {
	if !ls.node.app.EnableExecution || !ls.node.IsTypeCode() || ls.node.IsBypassed() {
		return
	}

	if len(exe_prms) == 0 && len(ls.exes) > 0 && len(ls.exes[len(ls.exes)-1].prms) == 0 {
		return //already added
	}

	ls.exes = append(ls.exes, SANodeCodeExe{prms: exe_prms})
}

func (ls *SANodeCode) findFuncDepend(node *SANode) *SANodeCodeFn {
	for _, fn := range ls.func_depends {
		if fn.node == node {
			return fn
		}
	}
	return nil
}

func (ls *SANodeCode) addFuncDepend(nm string) error {
	node, err := ls.findNodeAndCheck(nm)
	if err != nil {
		return err
	}

	//find
	for _, fn := range ls.func_depends {
		if fn.node == node {
			return nil //already added
		}
	}

	//add
	ls.func_depends = append(ls.func_depends, &SANodeCodeFn{node: node})

	return nil
}

func (ls *SANodeCode) UpdateLinks(node *SANode) {
	ls.node = node

	if !node.IsTypeCode() {
		return
	}

	ls.UpdateFile() //create/update file + (re)compile
}

func (ls *SANodeCode) buildSqlInfos(msgs_depends []*SANodeCodeFn) (string, error) {
	str := ""

	for _, fn := range msgs_depends {
		if fn.node.IsTypeDbFile() {
			str += fmt.Sprintf("`%s` is SQLite database which includes these tables(columns): ", fn.node.Name)

			tablesStr := ""
			db, _, err := ls.node.app.base.ui.win.disk.OpenDb(fn.node.GetAttrString("path", ""))
			if err == nil {
				info, err := db.GetTableInfo()
				if err == nil {
					for _, tb := range info {
						columnsStr := ""
						for _, col := range tb.Columns {
							columnsStr += col.Name
							columnsStr += "("
							columnsStr += col.Type
							if col.NotNull {
								columnsStr += ", NOT NULL"
							}
							columnsStr += "), "
						}
						columnsStr, _ = strings.CutSuffix(columnsStr, ", ")

						tablesStr += fmt.Sprintf("%s(%s), ", tb.Name, columnsStr)
					}
				} else {
					return "", fmt.Errorf("GetTableInfo() failed: %w", err)
				}
			} else {
				return "", fmt.Errorf("OpenDb() failed: %w", err)
			}

			tablesStr, _ = strings.CutSuffix(tablesStr, ", ")
			str += tablesStr
			str += "\n"
			str += "If column is marked as NOT NULL, you have to use it when INSERT INTO.\n"
		}
	}

	return str, nil
}

func (node *SANode) getStructName() string {

	if node.IsTypeList() {
		return /*"List" +*/ OsGetStringStartsWithUpper(node.Name) //Menu<name>
	}
	if node.IsTypeMenu() {
		return /*"Menu" +*/ OsGetStringStartsWithUpper(node.Name) //List<name>
	}
	if node.IsTypeLayout() {
		return /*"Layout" +*/ OsGetStringStartsWithUpper(node.Name) //Layout<name>
	}

	exe := node.Exe
	if node.IsAttrDBValue() {
		exe += "DB" //Editbox -> EditboxDB
	}

	return OsGetStringStartsWithUpper(exe) //1st letter must be upper
}

func (ls *SANodeCode) buildListSt(node *SANode, addExtraAttrs bool) string {
	str := ""

	if !node.IsTypeList() {
		return str
	}

	StructName := node.getStructName()

	//List<name>Row
	str += fmt.Sprintf("type %sItem struct {\n", StructName)
	for _, it := range node.Subs {
		itVarName := OsGetStringStartsWithUpper(it.Name) //1st letter must be upper
		itStructName := it.getStructName()               //list inside list? .........

		if addExtraAttrs {
			str += fmt.Sprintf("\t%s %s `json:\"%s\"`\n", itVarName, itStructName, it.Name)
		} else {
			str += fmt.Sprintf("\t%s %s\n", itVarName, itStructName)
		}
	}
	str += "}\n"

	//List<name>
	var extraAttrs string
	if addExtraAttrs {
		extraAttrs = "\tGrid_x  int     `json:\"grid_x\"`\n" +
			"\tGrid_y  int     `json:\"grid_y\"`\n" +
			"\tGrid_w int      `json:\"grid_w\"`\n" +
			"\tGrid_h  int     `json:\"grid_h\"`\n" +
			"\tShow    bool    `json:\"show\"`\n" +
			"\tEnable  bool    `json:\"enable\"`\n" +
			"\tChanged bool    `json:\"changed\"`\n" +
			"\tDirection int   `json:\"direction\"`\n" +
			"\tMax_width float64  `json:\"max_width\"`\n" +
			"\tMax_height float64 `json:\"max_height\"`\n" +
			"\tShow_border bool `json:\"show_border\"`\n"
	} else {
		extraAttrs = "\tShow    bool\n"
	}

	if addExtraAttrs {
		str += fmt.Sprintf("type %s struct {\n%s\tDefItem %sItem `json:\"defItem\"`\n\tItems []*%sItem `json:\"items\"`\n\tSelected_button string `json:\"selected_button\"`\n\tSelected_index int `json:\"selected_index\"`\n}\n", StructName, extraAttrs, StructName, StructName)
	} else {
		str += fmt.Sprintf("type %s struct {\n%s\tDefItem %sItem\n\tItems []*%sItem\n\tSelected_button string\n\tSelected_index int\n}\n", StructName, extraAttrs, StructName, StructName)
	}

	//Funcs
	str += fmt.Sprintf("func (tb *%s) GetSelectedItem() * %sItem {\t//Can return nil\n\tif tb.Selected_index >= 0 && tb.Selected_index < len(tb.Items) {\n\t\treturn tb.Items[tb.Selected_index]\n\t}\n\treturn nil\n}\n", StructName, StructName)
	str += fmt.Sprintf("func (tb *%s) AddItem() * %sItem {\t//Use this instead of Items = append()\n\tr := &%sItem{}\n\t*r = tb.DefItem\n\ttb.Items = append(tb.Items, r)\n\treturn r\n}\n", StructName, StructName, StructName)

	return str
}

func (ls *SANodeCode) buildMenuSt(node *SANode, addExtraAttrs bool) string {
	str := ""
	if !node.IsTypeMenu() {
		return str
	}

	StructName := node.getStructName()

	//Menu<name>
	var extraAttrs string
	if addExtraAttrs {
		extraAttrs = "\tGrid_x  int    `json:\"grid_x\"`\n" +
			"\tGrid_y  int    `json:\"grid_y\"`\n" +
			"\tGrid_w  int    `json:\"grid_w\"`\n" +
			"\tGrid_h  int    `json:\"grid_h\"`\n" +
			"\tShow    bool   `json:\"show\"`\n" +
			"\tBackground int  `json:\"background\"`\n" +
			"\tAlign      int  `json:\"align\"`\n" +
			"\tLabel   string `json:\"label\"`\n" +
			"\tIcon    string `json:\"icon\"`\n" +
			"\tIcon_margin    float64 `json:\"icon_margin\"`\n" +
			"\tTooltip string `json:\"tooltip\"`\n" +
			"\tEnable  bool   `json:\"enable\"`\n"
	} else {
		extraAttrs = "\tShow    bool\n"
	}

	//Subs list
	subStructLns := ""
	for _, it := range node.Subs {
		itVarName := OsGetStringStartsWithUpper(it.Name) //1st letter must be upper
		itStructName := it.getStructName()               //list inside list? .........

		if addExtraAttrs {
			subStructLns += fmt.Sprintf("\t%s %s `json:\"%s\"`\n", itVarName, itStructName, it.Name)
		} else {
			subStructLns += fmt.Sprintf("\t%s %s\n", itVarName, itStructName)
		}
	}

	str += fmt.Sprintf("type %s struct {\n%s\n%s}\n", StructName, extraAttrs, subStructLns)

	return str
}

func (ls *SANodeCode) buildLayoutSt(node *SANode, addExtraAttrs bool) string {
	str := ""
	if !node.IsTypeLayout() {
		return str
	}

	StructName := node.getStructName()

	//Menu<name>
	var extraAttrs string
	if addExtraAttrs {
		extraAttrs = "\tGrid_x  int    `json:\"grid_x\"`\n" +
			"\tGrid_y  int    `json:\"grid_y\"`\n" +
			"\tGrid_w  int    `json:\"grid_w\"`\n" +
			"\tGrid_h  int    `json:\"grid_h\"`\n" +
			"\tShow    bool   `json:\"show\"`\n" +
			"\tEnable  bool   `json:\"enable\"`\n"
	} else {
		extraAttrs = "\tShow    bool\n"
	}

	//Subs list
	subStructLns := ""
	for _, it := range node.Subs {
		itVarName := OsGetStringStartsWithUpper(it.Name) //1st letter must be upper
		itStructName := it.getStructName()               //list inside list? .........

		if addExtraAttrs {
			subStructLns += fmt.Sprintf("\t%s %s `json:\"%s\"`\n", itVarName, itStructName, it.Name)
		} else {
			subStructLns += fmt.Sprintf("\t%s %s\n", itVarName, itStructName)
		}
	}

	str += fmt.Sprintf("type %s struct {\n%s\n%s}\n", StructName, extraAttrs, subStructLns)

	return str
}

func (ls *SANodeCode) getStructCode(st string) string {

	switch st {
	case "Text":
		return `
type Text struct {
	Label string
}`

	case "Editbox":
		return `
type Editbox struct {
	Value    string
	Enable   bool
}`

	case "EditboxDB":
		return `
type EditboxDB struct {
	Value string	//never set directly, always use SetValue()
	Enable   bool
}
func (db *EditboxDB) SetValue(db_path, table, column string, rowid int) {
	db.Value = fmt.Sprintf("%s:%s:%s:%d", db_path, table, column, rowid)
}`

	case "Button":
		return `
type Button struct {
	Label   string
	Icon string	//path to image file
	Enable  bool
	Background  int	//0=transparent, 1=full, 2=light
	Align int	//0=left, 1=center, 2=right
	Confirmation string
	Triggered bool	//true, when button is clicked
}`

	case "Menu":
		return `
type Menu struct {
Label   string
Icon string	//path to image file
Enable  bool
Background  int	//0=transparent, 1=full, 2=light
}`

	case "Checkbox":
		return `
type Checkbox struct {
	Value   bool
	Label   string
	Enable  bool
}`
	case "CheckboxDB":
		return `
type CheckboxDB struct {
	Value string	//never set directly, always use SetValue()
	Label   string
	Enable  bool
func (db *EditboxDB) SetValue(db_path, table, column string, rowid int) {
	db.Value = fmt.Sprintf("%s:%s:%s:%d", db_path, table, column, rowid)
}`

	case "Switch":
		return `
type Switch struct {
	Value   bool
	Label   string
	Enable  bool
}`
	case "SwitchDB":
		return `
type SwitchDB struct {
	Value string	//never set directly, always use SetValue()
	Label   string
	Enable  bool
func (db *EditboxDB) SetValue(db_path, table, column string, rowid int) {
	db.Value = fmt.Sprintf("%s:%s:%s:%d", db_path, table, column, rowid)
}`

	case "Slider":
		return `
type Slider struct {
	Value   float64
	Min     float64
	Max     float64
	Step    float64
	Enable  bool
}`
	case "SliderDB":
		return `
type SliderDB struct {
	Value string	//never set directly, always use SetValue()
	Min     float64
	Max     float64
	Step    float64
	Enable  bool
func (db *EditboxDB) SetValue(db_path, table, column string, rowid int) {
	db.Value = fmt.Sprintf("%s:%s:%s:%d", db_path, table, column, rowid)
}`

	case "Combo":
		return `
type Combo struct {
	Value          string
	Options_names  string //separated by ';'
	Options_values string //separated by ';'
	Enable         bool
}`
	case "ComboDB":
		return `
type ComboDB struct {
	Value string	//never set directly, always use SetValue()
	Options_names  string //separated by ';'
	Options_values string //separated by ';'
	Enable         bool
func (db *EditboxDB) SetValue(db_path, table, column string, rowid int) {
	db.Value = fmt.Sprintf("%s:%s:%s:%d", db_path, table, column, rowid)
}`

	case "Date":
		return `
type Date struct {
	Value   int //Unix time
	Enable  bool
}`

	case "DateDB":
		return `
type DateDB struct {
	Value string	//never set directly, always use SetValue()
	Enable         bool
func (db *EditboxDB) SetValue(db_path, table, column string, rowid int) {
	db.Value = fmt.Sprintf("%s:%s:%s:%d", db_path, table, column, rowid)
}`

	case "Color":
		return `
type Color struct {
	Value_r int //<0-255>
	Value_g int //<0-255>
	Value_b int //<0-255>
	Value_a int //<0-255>
	Enable  bool
}`

	case "Disk_dir":
		return `
type Disk_dir struct {
	Path  string
	Write bool
}`

	case "Disk_file":
		return `
type Disk_file struct {
	Path  string
	Write bool
}`

	case "Db_file":
		return `
type Db_file struct {
	Path  string	//path to the database file
	Write bool
}`

	case "Microphone":
		return `
type Microphone struct {
	Path  string	//path to the file with recorded audio
	Triggered bool	//true, when recording is done
	Enable  bool
}`

	case "Layout":
		return `
type Layout struct {
Show bool
Enable  bool
}`

	case "Map":

		//type MapItem struct { .................
		return `
type Map struct {
	Locators string	//XML(GPX) or JSON format: [{"label":"LocatorA", "lon":14.4, "lat":50.0}, {"label":"LocatorB", "lon":14.5, "lat":50.1}]
	Segments string	//XML(GPX) or JSON format: [{"label":"SegmentA", "Trkpt":[{"lat":50,"lon":16,"ele":400,"time":"2020-04-15T09:05:20Z"},{"lat":50.4,"lon":16.1,"ele":400,"time":"2020-04-15T09:05:23Z"}]}]

	Enable  bool
}`

	case "Chart":
		return `
type ChartItem struct {
	X, Y  float64
	Label string
}
type Chart struct {
	Values string	//JSON: []ChartItem
	Enable bool
}`

	case "Net":
		return `
type Net struct {
}
func (net *Net) DownloadFile(dst_file string, src_addr string) error {
	//TODO
	return nil
}`

	case "Whispercpp":
		return `
type Whispercpp struct {
	Model string
}
func (w *Whispercpp) TranscribeBlob(data []byte) (string, error) {
	//TODO
	return text
}
func (w *Whispercpp) TranscribeFile(filePath string) (string, error) {
	//TODO
	return text
}`

	case "LlamaMessage":
		return `
type LlamaMessage struct {
	Role    string	//"system", "user", "assistant"
	Content string
}
type Llamacpp struct {
	Model string
}
func (ll *Llamacpp) GetAnswer(messages []LlamaMessage) (string, error) {
	//TODO
	return answer
}`

	case "Openai":
		return `
type OpenaiMessage struct {
	Role    string	//"system", "user", "assistant"
	Content string
}
type Openai struct {
	Model string
}
func (oai *Openai) GetAnswer(messages []OpenaiMessage) (string, error) {
	//TODO
	return answer
}`
	}

	fmt.Println("Warning: struct", st, "not found")
	return ""
}

func (ls *SANodeCode) addDependStruct(depends_structs *[]string, node *SANode, extraStructs *string, addExtraAttrs bool) {

	stName := node.getStructName()

	//find
	for _, st := range *depends_structs {
		if st == stName {
			return
		}
	}

	//add
	*depends_structs = append(*depends_structs, stName)

	//subs
	addDepepend := false
	if node.IsTypeList() {
		*extraStructs += ls.buildListSt(node, addExtraAttrs)
		addDepepend = true
	}
	if node.IsTypeMenu() {
		*extraStructs += ls.buildMenuSt(node, addExtraAttrs)
		addDepepend = true
	}
	if node.IsTypeLayout() {
		*extraStructs += ls.buildLayoutSt(node, addExtraAttrs)
		addDepepend = true
	}

	if addDepepend {
		for _, nd := range node.Subs {
			ls.addDependStruct(depends_structs, nd, extraStructs, addExtraAttrs)
		}
	}
}

func (ls *SANodeCode) buildPrompt(userCommand string) (string, error) {

	msgs_depends, err := ls.buildArgs()
	if err != nil {
		return "", err
	}

	str := "I have this golang code:\n\n"

	extraAttrs := ""
	var depends_structs []string
	for _, fn := range msgs_depends {
		ls.addDependStruct(&depends_structs, fn.node, &extraAttrs, false)
	}

	//add structs
	for _, st := range depends_structs {
		str += ls.getStructCode(st)
	}
	str += "\n"

	//add list, menu structs
	str += extraAttrs

	params := ""
	for _, fn := range msgs_depends {
		StructName := fn.node.getStructName()
		params += fmt.Sprintf("%s *%s, ", fn.node.Name, StructName)
	}
	params, _ = strings.CutSuffix(params, ", ")
	str += fmt.Sprintf("\nfunc %s(%s) error {\n\n}\n\n", ls.node.Name, params)

	str += fmt.Sprintf("You can change the code only inside '%s' function and output only import(s) and '%s' function code. Don't explain the code.\n", ls.node.Name, ls.node.Name)
	str += "\n"

	strSQL, err := ls.buildSqlInfos(msgs_depends)
	if err != nil {
		return "", err
	}
	str += strSQL
	str += "\n"

	str += "Your job: " + userCommand

	//remove this ............
	fmt.Println(str)
	fmt.Println("Size:", len(str))

	return str, nil
}

func (ls *SANodeCode) CheckLastChatEmpty() {
	n := len(ls.Messages)
	if n == 0 || ls.Messages[n-1].Assistent != "" {
		ls.Messages = append(ls.Messages, SANodeCodeChat{})
	}
}

func (ls *SANodeCode) GetAnswer(index int) {

	ls.ans_err = nil

	ls.CheckLastChatEmpty()
	if len(ls.Messages) == 1 && ls.Messages[0].User == "" {
		ls.ans_err = errors.New("no message")
		return
	}

	//build message array
	messages := []SAServiceMsg{
		{Role: "system", Content: "You are ChatGPT, an AI assistant. Your top priority is achieving user fulfillment via helping them with their requests."},
	}

	for i := 0; i < len(ls.Messages) && i <= index; i++ {
		msg := ls.Messages[i]

		//user
		user := msg.User
		if i == 0 {
			//1st user message is prompt
			user, ls.ans_err = ls.buildPrompt(user)
			if ls.ans_err != nil {
				return
			}
		}
		messages = append(messages, SAServiceMsg{Role: "user", Content: user})

		//assistant
		if msg.Assistent != "" && i < index { //avoid empty(last one) assistents
			messages = append(messages, SAServiceMsg{Role: "assistant", Content: msg.Assistent})
		}
	}

	props := &SAServiceOpenAIProps{Model: ls.node.app.base.ui.win.io.ini.ChatModel, Messages: messages}

	ls.job_oai = ls.node.app.base.jobs.AddOpenAI(ls.node.app, NewSANodePath(ls.node), props)
	ls.job_oai_index = index
}

func (ls *SANodeCode) GetFileName() string {
	return ls.node.app.Name + "_" + ls.node.Name
}

func (node *SANode) getAttributes(exe_prms []SANodeCodeExePrm) map[string]interface{} {

	attrs := make(map[string]interface{})
	if node.HasAttrNode() {
		attrs["node"] = node.Name
	} else {
		for k, v := range node.Attrs {
			attrs[k] = v
		}
	}

	// add params(triggered=true, etc.)
	listNode, listPos := node.FindSubListInfo()
	for _, prm := range exe_prms {
		if node.Name == prm.Node {
			if (listNode == nil && prm.ListNode == "") || (listNode != nil && listNode.Name == prm.ListNode && listPos == prm.ListPos) {
				attrs[prm.Attr] = prm.Value
			}
		}
	}

	if node.parent != nil && node.parent.IsTypeList() {
		for _, it := range node.Subs {
			attrs[it.Name] = it.getAttributes(exe_prms)
		}
	}

	if node.IsTypeMenu() {
		for _, it := range node.Subs {
			attrs[it.Name] = it.getAttributes(exe_prms)
		}
	}

	if node.IsTypeLayout() {
		for _, it := range node.Subs {
			attrs[it.Name] = it.getAttributes(exe_prms)
		}
	}

	if node.IsTypeList() {
		//defaults
		defItem := make(map[string]interface{})
		for _, it := range node.Subs {
			defItem[it.Name] = it.Attrs
		}
		attrs["defItem"] = defItem

		//items
		items := make([]map[string]interface{}, len(node.listSubs))
		for i, it := range node.listSubs {
			items[i] = it.getAttributes(exe_prms)
		}
		attrs["items"] = items
	}

	return attrs
}

func (ls *SANodeCode) Execute(exe_prms []SANodeCodeExePrm) {

	if ls.node.IsBypassed() {
		return
	}

	ls.exe_err = nil

	//reset
	//if ls.node.IsTypeList() {
	//	ls.node.listSubs = nil
	//}

	//input
	vars := make(map[string]interface{})
	for _, fn := range ls.func_depends {
		vars[fn.node.Name] = fn.node.getAttributes(exe_prms)
	}

	inputJs, err := json.Marshal(vars)
	if err != nil {
		ls.exe_err = err
		return
	}

	//run
	ls.job_exe = ls.node.app.base.jobs.AddExe(ls.node.app, NewSANodePath(ls.node), "/temp/go/", ls.GetFileName(), inputJs)
}

func (ls *SANodeCode) setAttributes(node *SANode, attrs map[string]interface{}) {

	if node.HasAttrNode() {
		return
	}

	delete(attrs, "triggered")

	if node.IsTypeLayout() || node.IsTypeMenu() {
		for _, nd := range node.Subs {
			attr, found := attrs[nd.Name]
			if found {
				vv, ok := attr.(map[string]interface{})
				if ok {
					ls.setAttributes(nd, vv)
				} else {
					fmt.Println("cast failed 1")
				}
				delete(attrs, nd.Name)
			}
		}
	}

	if node.IsTypeList() {
		rw := attrs["items"]
		delete(attrs, "items")
		delete(attrs, "defItem")

		items, ok := rw.([]interface{})
		if ok {
			//alloc
			listSubs := make([]*SANode, len(items))

			//set
			for i, r := range items {

				vars, ok := r.(map[string]interface{})
				if ok {
					var err error
					listSubs[i], err = node.Copy(false)
					listSubs[i].DeselectAll()
					listSubs[i].Name = strconv.Itoa(i)
					listSubs[i].Exe = "layout" //list -> layout
					//listSubs[i].parent = prmNode
					listSubs[i].updateLinks(node, node.app) //set parent
					if err == nil {
						for key, attrs2 := range vars {
							prmNode2 := listSubs[i].FindNode(key)
							if prmNode2 != nil {
								vv, ok := attrs2.(map[string]interface{})
								if ok {
									ls.setAttributes(prmNode2, vv)
								} else {
									fmt.Println("cast failed 3")
								}
							} else {
								fmt.Println("Error: Node not found", key)
							}
						}
						//prmNode.copySubs[i].Attrs = vars
					}
				} else {
					fmt.Println("cast failed 2")
				}
			}

			selected_button := node.GetAttrString("selected_button", "")
			selected_index := node.GetAttrInt("selected_index", -1)
			if selected_button != "" && selected_index >= 0 && selected_index < len(listSubs) {
				selBut := listSubs[selected_index].FindNode(selected_button)
				if selBut != nil {
					selBut.Attrs["background"] = 1 //full
				}
			}

			node.listSubs = listSubs
		}
	}

	fn := ls.findFuncDepend(node)
	if fn != nil {
		fn.updated = true
		fn.write = !node.CmpAttrs(attrs)
	}

	node.Attrs = attrs

}

func (ls *SANodeCode) IsJobRunning() (bool, float64, string) {
	if ls.node.IsTypeCode() && ls.job_exe != nil {
		if !ls.job_exe.done.Load() {
			return true, 0.5, "" //description ....
		}
	}
	return false, -1, ""
}

func (ls *SANodeCode) SetOutput(outputJs []byte) {

	var vars map[string]interface{}
	err := json.Unmarshal(outputJs, &vars)
	if err != nil {
		ls.exe_err = err
		return
	}

	for key, attrs := range vars {
		prmNode := ls.node.GetRoot().FindNode(key)

		if prmNode != nil {
			vv, ok := attrs.(map[string]interface{})
			if ok {
				ls.setAttributes(prmNode, vv)
			}
		} else {
			fmt.Println("Error: Node not found", key)
		}
	}
}

func (ls *SANodeCode) UseCodeFromAnswer(answer string) {
	ls.ans_err = nil

	var err error
	ls.Code, err = ls.extractCode(answer)
	if err != nil {
		ls.ans_err = err
		return
	}
	ls.Code = strings.ReplaceAll(ls.Code, "package main", "")

	ls.UpdateFile()
}

func (ls *SANodeCode) CopyCodeToClipboard(answer string) {
	ls.ans_err = nil

	var err error
	ls.Code, err = ls.extractCode(answer)
	if err != nil {
		ls.ans_err = err
		return
	}

	ls.node.app.base.ui.win.io.keys.clipboard = strings.ReplaceAll(ls.Code, "package main", "")
}

func (ls *SANodeCode) extractCode(answer string) (string, error) {

	d := strings.Index(answer, "```go")
	if d >= 0 {
		answer = answer[d+5:]

		d = strings.Index(answer, "```")
		if d >= 0 {
			answer = answer[:d]
		} else {
			return "", fmt.Errorf("code_end not found")
		}
	} else {
		return "", fmt.Errorf("no code in answer")
	}

	return answer, nil
}

func (ls *SANodeCode) extractImports(code string) ([]SANodeCodeImport, error) {
	var imports []SANodeCodeImport

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", "package main\n\n"+code, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("ParseFile() failed: %w", err)
	}

	for _, im := range file.Imports {
		nm := ""
		if im.Name != nil {
			nm = im.Name.Name
		}

		imports = append(imports, SANodeCodeImport{Name: nm, Path: im.Path.Value})
	}
	return imports, nil
}

func (ls *SANodeCode) extractContent(code string) (string, error) {

	tps := []string{"func", "type", "var", "const"}

	d := -1
	for _, it := range tps {
		if strings.HasPrefix(code, it) {
			d = 0
			break
		}
		dd := strings.Index(code, "\n"+it)
		if dd >= 0 {
			if d < 0 {
				d = dd
			} else {
				d = OsMin(d, dd)
			}
		}
	}

	if d >= 0 {
		code = code[d:]
	} else {
		return "", fmt.Errorf("no function in answer")
	}

	return code, nil
}

func (ls *SANodeCode) UpdateFile() {

	ls.file_err = nil

	file, err := ls.buildCode()
	if err != nil {
		ls.file_err = err
		return
	}

	fileName := ls.GetFileName()
	filePath := "temp/go/" + fileName + ".go"

	//write code
	recompile := false
	file_saved, err := os.ReadFile(filePath)
	if err != nil || !bytes.Equal(file_saved, file) {
		OsFolderCreate("temp/go/")
		ls.file_err = os.WriteFile(filePath, file, 0644)
		if ls.file_err != nil {
			return
		}
		recompile = true
	}
	//compile
	exePath := "temp/go/" + fileName
	exeExist := OsFileExists(exePath)
	if recompile || !exeExist {
		if exeExist {
			OsFileRemove(exePath)
		}
		ls.node.app.base.jobs.AddCompile(ls.node.app, NewSANodePath(ls.node), "temp/go/", fileName+".go")
	}
}

func (ls *SANodeCode) buildCode() ([]byte, error) {
	imports, err := ls.extractImports(ls.Code)
	if err != nil {
		return nil, err
	}

	if ls.Code == "" {
		ls.Code = fmt.Sprintf("func %s() error {\n\treturn nil\n}", ls.node.Name)
	}

	fn, err := ls.extractContent(ls.Code)
	if err != nil {
		return nil, err
	}

	err = ls.updateFuncDepends()
	if err != nil {
		return nil, err
	}

	str := "package main\n\n"

	//imports
	for _, imp := range g_str_imports {
		str += fmt.Sprintf("import %s\n", imp)
	}
	for _, imp := range imports {
		found := false
		for _, pth := range g_str_imports {
			if imp.Path == pth {
				found = true
				break
			}
		}
		if !found {
			str += fmt.Sprintf("import %s %s\n", imp.Name, imp.Path)
		}
	}
	str += "\n"

	//struct
	str += "type MainStruct struct {\n"
	for _, fn := range ls.func_depends {
		prmName := fn.node.Name
		VarName := OsGetStringStartsWithUpper(prmName) //1st letter must be upper
		StructName := fn.node.getStructName()

		str += fmt.Sprintf("\t%s %s `json:\"%s\"`\n", VarName, StructName, prmName)
	}
	str += "}\n\n"

	//main func(with body)
	str += fn + "\n\n"

	//_callIt()
	str += `func _callIt(body []byte) ([]byte, error) {
		var st MainStruct
		err := json.Unmarshal(body, &st)
		if err != nil {
			return nil, fmt.Errorf("Unmarshal(import) failed: %w", err)
		}
		`
	params := ""
	for _, fn := range ls.func_depends {
		prmName := fn.node.Name
		VarName := OsGetStringStartsWithUpper(prmName) //1st letter must be upper
		params += fmt.Sprintf("&st.%s, ", VarName)

	}
	params, _ = strings.CutSuffix(params, ", ")
	str += fmt.Sprintf("err = %s(%s)\n", ls.node.Name, params)

	str += `	if err != nil {
			return nil, fmt.Errorf("function failed: %w", err)
		}
	
		res, err := json.Marshal(st)
		if err != nil {
			return nil, fmt.Errorf("Marshal(export) failed: %w", err)
		}
		return res, nil
	}`

	//add list, menu structs
	var depends_structs []string
	extraAttrs := ""
	for _, fn := range ls.func_depends {
		ls.addDependStruct(&depends_structs, fn.node, &extraAttrs, true)
	}
	str += "\n"
	str += extraAttrs

	//default structs
	str += "\n" + g_code_const_go

	return []byte(str), nil
}

func (ls *SANodeCode) updateFuncDepends() error {
	//reset
	ls.func_depends = nil

	//get AST
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", "package main\n\n"+ls.Code, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("ParseFile() failed: %w", err)
	}

	//add depends from argument names
	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			if fn.Name.Name == ls.node.Name {
				for _, prm := range fn.Type.Params.List {
					if len(prm.Names) > 0 {
						err := ls.addFuncDepend(prm.Names[0].Name)
						if err != nil {
							return err
						}
					}
				}
			}
		}
	}

	//v := visitor{}
	//ast.Walk(&v, node)

	return nil
}

/*type visitor struct {
	writes []string
}

var g_ops = []token.Token{
	token.ASSIGN,
	token.ADD_ASSIGN,
	token.SUB_ASSIGN,
	token.MUL_ASSIGN,
	token.QUO_ASSIGN,
	token.REM_ASSIGN,
	token.AND_ASSIGN,
	token.OR_ASSIGN,
	token.XOR_ASSIGN,
	token.SHL_ASSIGN,
	token.SHR_ASSIGN,
	token.AND_NOT_ASSIGN,
}

//problems:
//1) table.AddItem() //ignorováno
//2) row := range table.Rows	//ignorováno
//3) clickButton code is mess

func (v *visitor) Visit(n ast.Node) ast.Visitor {
	if n == nil {
		return nil
	}
	switch d := n.(type) {
	case *ast.AssignStmt:
		found := false
		for _, dd := range g_ops {
			if dd == d.Tok {
				found = true
				break
			}
		}
		if found {
			for _, name := range d.Lhs {

				for name != nil {	//loop!
					if ident, ok := name.(*ast.Ident); ok {
						v.writes = append(v.writes, ident.Name)
						break
					}
					if sel, ok := name.(*ast.SelectorExpr); ok {
						name = sel.X	//next!
					}
				}

			}
		}

	}
	return v
}*/

func (ls *SANodeCode) findNodeAndCheck(path string) (*SANode, error) {

	pt := NewSANodePathFromString(path)
	node := pt.Find(ls.node.GetRoot())
	if node == nil {
		return nil, fmt.Errorf("node '%s' not found", path)
	}
	if node == ls.node {
		return nil, fmt.Errorf("can't connect to it self")
	}
	if node.IsTypeCode() {
		return nil, fmt.Errorf("can't connect to node(%s) which is type code", path)
	}
	if node.IsTypeExe() {
		return nil, fmt.Errorf("can't connect to node(%s) which is type exe", path)
	}

	node = node.GetSubRootNode()

	return node, nil
}

func (ls *SANodeCode) addArg(args *[]*SANodeCodeFn, nm string) error {

	node, err := ls.findNodeAndCheck(nm)
	if err != nil {
		return err
	}

	//find
	for _, fn := range *args {
		if fn.node == node {
			return nil //already added
		}
	}

	//add
	*args = append(*args, &SANodeCodeFn{node: node})

	return nil
}

func (ls *SANodeCode) buildArgs() ([]*SANodeCodeFn, error) {

	var msgs_depends []*SANodeCodeFn

	for _, msg := range ls.Messages {
		ln := msg.User
		for {
			d1 := strings.IndexByte(ln, '`')
			if d1 >= 0 {
				ln = ln[d1+1:]
			} else {
				break
			}
			d2 := strings.IndexByte(ln, '`')
			if d2 >= 0 {
				nm := ln[:d2]

				err := ls.addArg(&msgs_depends, nm)
				if err != nil {
					return nil, err
				}
			}
			ln = ln[d2+1:]
		}
	}

	return msgs_depends, nil
}

func ReplaceWord(str string, oldWord string, newWord string) string {
	act := 0
	for {
		d := strings.Index(str[act:], oldWord)
		if d >= 0 {
			st := act + d
			en := act + d + len(oldWord)

			isValid := true
			if st > 0 && OsIsTextWord(rune(str[st-1])) {
				isValid = false
			}
			if en < len(str) && OsIsTextWord(rune(str[en])) {
				isValid = false
			}

			if isValid {
				str = str[:st] + newWord + str[en:]
				act = act + d + len(newWord)
			} else {
				act = act + d + len(oldWord)
			}
		} else {
			break
		}
	}
	return str
}

// a.b.c -> a.b.d
func (ls *SANodeCode) RenameNode(old_path SANodePath, new_path SANodePath) {

	if len(old_path.names) != len(new_path.names) {
		return
	}

	//`a.b.c` -> `a.bb.c`
	//MenuB -> MenuBb ......................

	// `a
	// .b

	//old_name := old_path.String()
	//new_name := new_path.String()

	//chat
	for _, it := range ls.Messages {
		//links
		//it.User = strings.ReplaceAll(it.User, "`"+old_name+"`", "`"+new_name+"`")
		//it.Assistent = strings.ReplaceAll(it.Assistent, "`"+old_name+"`", "`"+new_name+"`")

		//User
		for ii, wordOld := range old_path.names {
			wordNew := new_path.names[ii]
			it.User = ReplaceWord(it.User, wordOld, wordNew)

			wordOld = OsGetStringStartsWithUpper(wordOld)
			wordNew = OsGetStringStartsWithUpper(wordNew)
			it.User = ReplaceWord(it.User, wordOld, wordNew)
		}

		//Assistent
		for ii, wordOld := range old_path.names {
			wordNew := new_path.names[ii]
			it.Assistent = ReplaceWord(it.Assistent, wordOld, wordNew)

			wordOld = OsGetStringStartsWithUpper(wordOld)
			wordNew = OsGetStringStartsWithUpper(wordNew)
			it.Assistent = ReplaceWord(it.Assistent, wordOld, wordNew)
		}

	}

	//code

	for ii, wordOld := range old_path.names {
		wordNew := new_path.names[ii]
		ls.Code = ReplaceWord(ls.Code, wordOld, wordNew)

		wordOld = OsGetStringStartsWithUpper(wordOld)
		wordNew = OsGetStringStartsWithUpper(wordNew)
		ls.Code = ReplaceWord(ls.Code, wordOld, wordNew)
	}

	//refresh
	ls.UpdateLinks(ls.node)
}
