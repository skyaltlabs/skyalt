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

	ArgsProps []*SANodeCodeArg

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

func (ls *SANodeCode) findNodeName(nm string) (*SANode, error) {
	found := ls.node.FindNode(nm)
	if found == nil {
		return nil, fmt.Errorf("node '%s' not found", nm)
	}
	if found == ls.node && !found.IsTypeList() {
		return nil, fmt.Errorf("can't connect to it self")
	}
	if found.IsTypeCode() && !found.IsTypeList() {
		return nil, fmt.Errorf("can't connect to node(%s) which is type code", nm)
	}
	return found, nil
}

func (ls *SANodeCode) AddExe(prms []SANodeCodeExePrm) {
	if !ls.node.app.EnableExecution || !ls.node.IsTypeCode() || ls.node.IsBypassed() {
		return
	}

	if len(prms) == 0 && len(ls.exes) > 0 && len(ls.exes[len(ls.exes)-1].prms) == 0 {
		return //already added
	}

	ls.exes = append(ls.exes, SANodeCodeExe{prms: prms})
}

func (ls *SANodeCode) findFuncDepend(node *SANode) *SANodeCodeFn {
	for _, fn := range ls.func_depends {
		if fn.node == node {
			return fn
		}
	}
	return nil
}

func (ls *SANodeCode) GetArg(node string) *SANodeCodeArg {

	//find
	for _, arg := range ls.ArgsProps {
		if arg.Node == node {
			return arg
		}
	}

	//add
	ls.ArgsProps = append(ls.ArgsProps, &SANodeCodeArg{Node: node})

	return ls.ArgsProps[len(ls.ArgsProps)-1]
}

func (ls *SANodeCode) addFuncDepend(nm string) error {
	node, err := ls.findNodeName(nm)
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

func (ls *SANodeCode) addMsgDepend(msgs_depends *[]*SANodeCodeFn, nm string) error {
	node, err := ls.findNodeName(nm)
	if err != nil {
		return err
	}

	//find
	for _, fn := range *msgs_depends {
		if fn.node == node {
			return nil //already added
		}
	}

	//add
	*msgs_depends = append(*msgs_depends, &SANodeCodeFn{node: node})

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
			str += fmt.Sprintf("'%s' is SQLite database which includes these tables(columns): ", fn.node.Name)

			tablesStr := ""
			db, _, err := ls.node.app.base.ui.win.disk.OpenDb(fn.node.GetAttrString("path", ""))
			if err == nil {
				info, err := db.GetTableInfo()
				if err == nil {
					for _, tb := range info {
						columnsStr := ""
						for _, col := range tb.Columns {
							columnsStr += col.Name + "(" + col.Type + "), "
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
		}
	}

	return str, nil
}

func (node *SANode) getStructName() string {

	if node.IsTypeList() {
		return "List" + strings.ToUpper(node.Name[0:1]) + node.Name[1:] //List<name>
	}

	exe := node.Exe
	if node.IsAttrDBValue() {
		exe += "DB" //Editbox -> EditboxDB
	}

	return strings.ToUpper(exe[0:1]) + exe[1:] //1st letter must be upper
}

func (ls *SANodeCode) buildListStructs(depends []*SANodeCodeFn, addExtraAttrs bool) string {
	str := ""
	for _, fn := range depends {
		if !fn.node.IsTypeList() {
			continue
		}

		StructName := fn.node.getStructName()

		//List<name>Row
		str += fmt.Sprintf("type %sRow struct {\n", StructName)
		for _, it := range fn.node.Subs {
			itVarName := strings.ToUpper(it.Name[0:1]) + it.Name[1:] //1st letter must be upper
			itStructName := it.getStructName()                       //list inside list? .........

			str += fmt.Sprintf("\t%s %s `json:\"%s\"`\n", itVarName, itStructName, it.Name)
		}
		str += "}\n"

		//List<name>
		extraAttrs := "\tGrid_x  int     `json:\"grid_x\"`\n" +
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
		if !addExtraAttrs {
			extraAttrs = ""
		}

		str += fmt.Sprintf("type %s struct {\n%s\tDefRow %sRow `json:\"defRow\"`\n\tRows []*%sRow `json:\"rows\"`\n\tSelected_button string `json:\"selected_button\"`\n\tSelected_index int `json:\"selected_index\"`\n}\n", StructName, extraAttrs, StructName, StructName)

		//Funcs
		str += fmt.Sprintf("func (tb *%s) GetSelectedRow() * %sRow {\t//Can return nil\n\tif tb.Selected_index >= 0 && tb.Selected_index < len(tb.Rows) {\n\t\treturn tb.Rows[tb.Selected_index]\n\t}\n\treturn nil\n}\n", StructName, StructName)
		str += fmt.Sprintf("func (tb *%s) AddRow() * %sRow {\t//Use this instead of Rows = append()\n\tr := &%sRow{}\n\t*r = tb.DefRow\n\ttb.Rows = append(tb.Rows, r)\n\treturn r\n}\n", StructName, StructName, StructName)
	}
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
	Confirmation string
	Background  int	//0=transparent, 1=full, 2=light
	Triggered bool	//true, when button is clicked
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

func (ls *SANodeCode) addDependStruct(depends_structs *[]string, nm string) {
	//find
	for _, st := range *depends_structs {
		if st == nm {
			return
		}
	}

	//add
	*depends_structs = append(*depends_structs, nm)
}

func (ls *SANodeCode) buildPrompt(userCommand string) (string, error) {

	msgs_depends, err := ls.buildMsgsDepends()
	if err != nil {
		return "", err
	}

	str := "I have this golang code:\n\n"

	//str += g_code_const_gpt + "\n"
	var depends_structs []string
	for _, fn := range msgs_depends {

		if fn.node.IsTypeList() {
			for _, nd := range fn.node.Subs {
				StructName := nd.getStructName()
				ls.addDependStruct(&depends_structs, StructName)
			}
		}

		StructName := fn.node.getStructName()
		ls.addDependStruct(&depends_structs, StructName)
	}

	//add structs
	for _, st := range depends_structs {
		str += ls.getStructCode(st)
	}
	str += "\n"

	//add list structs
	str += ls.buildListStructs(msgs_depends, false)

	params := ""
	for _, fn := range msgs_depends {
		StructName := fn.node.getStructName()
		params += fmt.Sprintf("%s *%s, ", fn.node.Name, StructName)
	}
	params, _ = strings.CutSuffix(params, ", ")
	str += fmt.Sprintf("\nfunc %s(%s) error {\n\n}\n\n", ls.node.Name, params)

	str += fmt.Sprintf("You can change the code only inside '%s' function and output only import(s) and '%s' function code. Don't explain the code.\n", ls.node.Name, ls.node.Name)

	strSQL, err := ls.buildSqlInfos(msgs_depends)
	if err != nil {
		return "", err
	}
	str += strSQL

	str += "Your job: " + userCommand

	//comment ............
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
		prmNode := fn.node

		attrs := make(map[string]interface{})
		if prmNode.HasAttrNode() {
			attrs["node"] = prmNode.Name
		} else {
			for k, v := range prmNode.Attrs {
				attrs[k] = v
			}
		}

		//add params(triggered=true, etc.)
		for _, prm := range exe_prms {
			if prmNode.Name == prm.Node {
				if prm.ListNode == "" {
					attrs[prm.Attr] = prm.Value
				}
			}
		}

		if prmNode.IsTypeList() {
			//defaults
			defRows := make(map[string]interface{})
			for _, it := range prmNode.Subs {
				defRows[it.Name] = it.Attrs
			}
			attrs["defRow"] = defRows

			//rows
			rows := make([]map[string]interface{}, len(prmNode.listSubs))
			for i, it := range prmNode.listSubs {

				rws := make(map[string]interface{})
				for _, it := range it.Subs {

					itAttrs := make(map[string]interface{})
					for k, v := range it.Attrs {
						itAttrs[k] = v
					}

					//add params(triggered=true, etc.)
					for _, prm := range exe_prms {
						if prm.ListPos == i && prmNode.Name == prm.Node && it.Name == prm.ListNode {
							itAttrs[prm.Attr] = prm.Value
						}
					}

					rws[it.Name] = itAttrs
				}
				rows[i] = rws
			}
			attrs["rows"] = rows
		}

		vars[prmNode.Name] = attrs
	}

	inputJs, err := json.Marshal(vars)
	if err != nil {
		ls.exe_err = err
		return
	}

	//run
	ls.job_exe = ls.node.app.base.jobs.AddExe(ls.node.app, NewSANodePath(ls.node), "/temp/go/", ls.GetFileName(), inputJs)
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
		prmNode := ls.node.FindNode(key)

		if prmNode != nil {
			if !prmNode.HasAttrNode() {
				switch vv := attrs.(type) {
				case map[string]interface{}:
					delete(vv, "triggered")

					//read/write
					fn := ls.findFuncDepend(prmNode)
					if fn != nil {
						fn.updated = true
						fn.write = !prmNode.CmpAttrs(vv)
					}

					prmNode.Attrs = vv
				}
			}

			if prmNode.IsTypeList() {
				rw := prmNode.Attrs["rows"]
				delete(prmNode.Attrs, "rows")

				rows, ok := rw.([]interface{})
				if ok {
					//alloc
					listSubs := make([]*SANode, len(rows))

					//set
					for i, r := range rows {

						vars, ok := r.(map[string]interface{})
						if ok {
							listSubs[i], err = prmNode.Copy(false)
							listSubs[i].DeselectAll()
							listSubs[i].Name = strconv.Itoa(i)
							listSubs[i].Exe = "layout" //list -> layout
							//listSubs[i].parent = prmNode
							listSubs[i].updateLinks(prmNode, prmNode.app) //set parent
							if err == nil {
								for key, attrs := range vars {
									prmNode2 := listSubs[i].FindNodeSubs(key)
									if prmNode2 != nil {
										if !prmNode2.HasAttrNode() {
											switch vv := attrs.(type) {
											case map[string]interface{}:
												delete(vv, "triggered")
												prmNode2.Attrs = vv
											}
										}
									} else {
										fmt.Println("Error: Node not found", key)
									}
								}
								//prmNode.copySubs[i].Attrs = vars
							}
						}
					}

					/*diff := len(prmNode.listSubs) != len(listSubs)
					if !diff {
						for i, nd := range prmNode.listSubs {
							if !nd.CmpListSub(listSubs[i]) {
								diff = true
								break
							}
						}
					}*/

					selected_button := prmNode.GetAttrString("selected_button", "")
					selected_index := prmNode.GetAttrInt("selected_index", -1)
					if selected_button != "" && selected_index >= 0 && selected_index < len(listSubs) {
						selBut := listSubs[selected_index].FindNodeSubs(selected_button)
						if selBut != nil {
							selBut.Attrs["background"] = 1 //full
						}
					}

					prmNode.listSubs = listSubs
					//prmNode.ResetTriggers() //reset sub-buttons clicks

					//if diff {
					//	prmNode.SetChange(nil)
					//}
				}
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
	node, err := parser.ParseFile(fset, "test.go", "package main\n\n"+code, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("ParseFile() failed: %w", err)
	}

	for _, im := range node.Imports {
		nm := ""
		if im.Name != nil {
			nm = im.Name.Name
		}

		imports = append(imports, SANodeCodeImport{Name: nm, Path: im.Path.Value})
	}
	return imports, nil
}

func (ls *SANodeCode) extractContent(code string) (string, error) {
	d := strings.Index(code, "\nfunc")
	dd := strings.Index(code, "\ntype")
	if dd >= 0 && dd < d {
		d = dd
	}
	dd = strings.Index(code, "\nvar")
	if dd >= 0 && dd < d {
		d = dd
	}
	dd = strings.Index(code, "\nconst")
	if dd >= 0 && dd < d {
		d = dd
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
		VarName := strings.ToUpper(prmName[0:1]) + prmName[1:] //1st letter must be upper
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
		VarName := strings.ToUpper(prmName[0:1]) + prmName[1:] //1st letter must be upper
		params += fmt.Sprintf("&st.%s, ", VarName)

	}
	params, _ = strings.CutSuffix(params, ", ")
	str += fmt.Sprintf("err = %s(%s)\n", ls.node.Name, params)

	str += `	if err != nil {
			return nil, fmt.Errorf("Chat() failed: %w", err)
		}
	
		res, err := json.Marshal(st)
		if err != nil {
			return nil, fmt.Errorf("Marshal(export) failed: %w", err)
		}
		return res, nil
	}`

	//tables
	str += "\n"
	str += ls.buildListStructs(ls.func_depends, true)

	//rest
	str += "\n" + g_code_const_go

	return []byte(str), nil
}

func (ls *SANodeCode) updateFuncDepends() error {
	//reset
	ls.func_depends = nil

	//get AST
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, "test.go", "package main\n\n"+ls.Code, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("ParseFile() failed: %w", err)
	}

	//add depends from argument names
	for _, decl := range node.Decls {
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
//1) table.AddRow() //ignorováno
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

				for name != nil {
					if ident, ok := name.(*ast.Ident); ok {
						v.writes = append(v.writes, ident.Name)
						break
					}
					if sel, ok := name.(*ast.SelectorExpr); ok {
						name = sel.X
					}
				}

			}
		}

	}
	return v
}*/

func (ls *SANodeCode) buildMsgsDepends() ([]*SANodeCodeFn, error) {

	var msgs_depends []*SANodeCodeFn

	for _, msg := range ls.Messages {
		ln := msg.User
		for {
			d1 := strings.IndexByte(ln, '\'')
			if d1 >= 0 {
				ln = ln[d1+1:]
			} else {
				break
			}
			d2 := strings.IndexByte(ln, '\'')
			if d2 >= 0 {
				nm := ln[:d2]

				err := ls.addMsgDepend(&msgs_depends, nm)
				if err != nil {
					return nil, err
				}
			}
			ln = ln[d2+1:]
		}
	}

	return msgs_depends, nil
}

func IsWordLetter(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch == '_'
}
func ReplaceWord(str string, oldWord string, newWord string) string {
	act := 0
	for {
		d := strings.Index(str[act:], oldWord)
		if d >= 0 {
			st := act + d
			en := act + d + len(oldWord)

			isValid := true
			if st > 0 && IsWordLetter(str[st-1]) {
				isValid = false
			}
			if en < len(str) && IsWordLetter(str[en]) {
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

func (ls *SANodeCode) RenameNode(old_name string, new_name string) {

	//arguments
	for i, arg := range ls.ArgsProps {
		if arg.Node == old_name {
			ls.ArgsProps[i].Node = new_name
		}
	}

	//chat
	for _, it := range ls.Messages {
		it.User = strings.ReplaceAll(it.User, "'"+old_name+"'", "'"+new_name+"'")
		it.Assistent = strings.ReplaceAll(it.Assistent, "'"+old_name+"'", "'"+new_name+"'")

		it.Assistent = ReplaceWord(it.Assistent, old_name, new_name)
	}

	//func
	ls.Code = ReplaceWord(ls.Code, old_name, new_name)

	//refresh
	ls.UpdateLinks(ls.node)
}
