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
	"os/exec"
	"strconv"
	"strings"

	_ "embed"
)

//go:embed sa_node_const_go.goo
var g_code_const_go string

//go:embed sa_node_const_gpt.goo
var g_code_const_gpt string

var g_str_imports = []string{"\"bytes\"", "\"encoding/json\"", "\"fmt\"", "\"io\"", "\"net/http\"", "\"os\"", "\"strconv\""}

type SANodeCodeChat struct {
	User      string
	Assistent string
}
type SANodeCodeImport struct {
	Name string
	Path string
}
type SANodeCode struct {
	node *SANode

	Triggers []string //nodes
	Messages []SANodeCodeChat

	Code string

	func_depends []*SANode
	msgs_depends []*SANode

	output string //terminal

	exeTimeSec float64

	file_err error
	exe_err  error
	ans_err  error
}

func InitSANodeCode(node *SANode) SANodeCode {
	ls := SANodeCode{}
	ls.node = node
	return ls
}

func (ls *SANodeCode) addTrigger(name string) {
	for _, tr := range ls.Triggers {
		if tr == name {
			return
		}
	}
	ls.Triggers = append(ls.Triggers, name)
}

func (ls *SANodeCode) findNodeName(nm string) (*SANode, error) {
	node := ls.node.FindNode(nm)
	if node == nil {
		return nil, fmt.Errorf("node '%s' not found", nm)
	}
	if node == ls.node {
		return nil, fmt.Errorf("can't connect to it self")
	}
	if node.IsTypeCode() {
		return nil, fmt.Errorf("can't connect to node(%s) which is type code", node.Name)
	}
	return node, nil
}

func (ls *SANodeCode) addFuncDepend(nm string) error {
	node, err := ls.findNodeName(nm)
	if err != nil {
		return err
	}

	//find
	for _, dp := range ls.func_depends {
		if dp.Name == node.Name {
			return nil //already added
		}
	}

	//add
	ls.func_depends = append(ls.func_depends, node)

	return nil
}

func (ls *SANodeCode) addMsgDepend(nm string) error {
	node, err := ls.findNodeName(nm)
	if err != nil {
		return err
	}

	//find
	for _, dp := range ls.msgs_depends {
		if dp.Name == node.Name {
			return nil //already added
		}
	}

	//add
	ls.msgs_depends = append(ls.msgs_depends, node)

	return nil
}

func (ls *SANodeCode) UpdateLinks(node *SANode) {
	ls.node = node

	if !node.IsTypeCode() {
		return
	}

	ls.UpdateFile() //create/update file + (re)compile
}

func (ls *SANodeCode) buildSqlInfos() (string, error) {
	str := ""

	for _, prmNode := range ls.msgs_depends {
		if prmNode.Exe == "sqlite" {
			str += fmt.Sprintf("'%s' is SQLite database which includes these tables(columns): ", prmNode.Name)

			tablesStr := ""
			db, _, err := ls.node.app.base.ui.win.disk.OpenDb(prmNode.GetAttrString("path", ""))
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

func (ls *SANodeCode) buildPrompt(userCommand string) (string, error) {

	err := ls.updateMsgsDepends()
	if err != nil {
		return "", err
	}

	str := "I have this golang code:\n\n"
	str += g_code_const_gpt + "\n"

	params := ""
	for _, prmNode := range ls.msgs_depends {

		StructName := strings.ToUpper(prmNode.Exe[0:1]) + prmNode.Exe[1:] //1st letter must be upper
		params += fmt.Sprintf("%s *%s, ", prmNode.Name, StructName)
	}
	params, _ = strings.CutSuffix(params, ", ")
	str += fmt.Sprintf("\nfunc %s(%s) error {\n\n}\n\n", ls.node.Name, params)

	str += fmt.Sprintf("You can change the code only inside '%s' function and output only import(s) and '%s' function code. Don't explain the code.\n", ls.node.Name, ls.node.Name)

	strSQL, err := ls.buildSqlInfos()
	if err != nil {
		return "", err
	}
	str += strSQL

	str += "Your job: " + userCommand

	return str, nil
}

func (ls *SANodeCode) CheckLastChatEmpty() {
	n := len(ls.Messages)
	if n == 0 || ls.Messages[n-1].Assistent != "" {
		ls.Messages = append(ls.Messages, SANodeCodeChat{})
	}
}

func (ls *SANodeCode) GetAnswer() {

	ls.ans_err = nil

	ls.CheckLastChatEmpty()
	if ls.Messages[0].User == "" {
		ls.ans_err = errors.New("no message")
		return
	}

	messages := []SAServiceMsg{
		{Role: "system", Content: "You are ChatGPT, an AI assistant. Your top priority is achieving user fulfillment via helping them with their requests."},
	}

	for i, msg := range ls.Messages {
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
		if msg.Assistent != "" { //avoid empty(last one) assistents
			messages = append(messages, SAServiceMsg{Role: "assistant", Content: msg.Assistent})
		}
	}

	oai, err := ls.node.app.base.services.GetOpenAI()
	if err != nil {
		ls.ans_err = err
		return
	}

	answer, err := oai.Complete(&SAServiceOpenAIProps{Model: "gpt-4-turbo-preview", Messages: messages})
	if err != nil {
		ls.ans_err = err
		return
	}

	//save
	ls.Messages[len(ls.Messages)-1].Assistent = string(answer)
}

func (ls *SANodeCode) IsTriggered() bool {

	for _, tr := range ls.Triggers {
		nd := ls.node.FindNode(tr)
		if nd != nil {
			if nd.IsTriggered() {
				return true
			}
		} else {
			fmt.Println("Error: Node not found", tr)
		}
	}
	return false
}

func (ls *SANodeCode) GetFileName() string {
	return ls.node.app.Name + "_" + ls.node.Name
}

func (ls *SANodeCode) Execute() {
	st := OsTime()

	ls.exe_err = nil

	//input
	{
		vars := make(map[string]interface{})
		for _, prmNode := range ls.func_depends {
			vars[prmNode.Name] = prmNode.Attrs

			attrs := prmNode.Attrs
			if prmNode.HasAttrNode() {
				attrs = make(map[string]interface{})
				attrs["node"] = prmNode.Name

			}
			vars[prmNode.Name] = attrs
		}
		inputJs, err := json.Marshal(vars)
		if err != nil {
			ls.exe_err = err
			return
		}

		ls.node.app.base.services.SetJob(inputJs, ls.node.app)
	}

	//process
	{
		cmd := exec.Command("./temp/go/"+ls.GetFileName(), strconv.Itoa(ls.node.app.base.services.port))
		output, err := cmd.CombinedOutput()
		if err != nil {
			ls.exe_err = errors.New(string(output))
			return
		}
		ls.output = string(output)

		//možná dát chybu do codeErr a nakopírovat ji do node? .................................
	}

	//output
	{
		outputJs := ls.node.app.base.services.GetResult()

		var vars map[string]interface{}
		err := json.Unmarshal(outputJs, &vars)
		if err != nil {
			ls.exe_err = err
			return
		}

		for key, node := range vars {
			prmNode := ls.node.FindNode(key)
			if prmNode != nil {
				if !prmNode.HasAttrNode() {
					switch vv := node.(type) {
					case map[string]interface{}:
						prmNode.Attrs = vv
					}
				}
			} else {
				fmt.Println("Error: Node not found", key)
			}
		}

	}

	ls.exeTimeSec = OsTime() - st
	fmt.Printf("Executed node '%s' in %.3f\n", ls.node.Name, ls.exeTimeSec)
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

func (ls *SANodeCode) extractFunc(code string) (string, error) {

	d := strings.Index(code, "func "+ls.node.Name)
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
	if recompile || !OsFileExists("temp/go/"+fileName) {
		cmd := exec.Command("go", "build", fileName+".go")
		cmd.Dir = "temp/go/"

		output, err := cmd.CombinedOutput()
		if err != nil {
			ls.file_err = errors.New(string(output))
			return
		}
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

	fn, err := ls.extractFunc(ls.Code)
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
	for _, prmNode := range ls.func_depends {
		VvarName := strings.ToUpper(prmNode.Name[0:1]) + prmNode.Name[1:] //1st letter must be upper
		StructName := strings.ToUpper(prmNode.Exe[0:1]) + prmNode.Exe[1:] //1st letter must be upper
		str += fmt.Sprintf("\t%s %s `json:\"%s\"`\n", VvarName, StructName, prmNode.Name)
	}
	str += "}\n\n"

	//main func(with body)
	str += fn + "\n\n"

	//_callIt()
	str += `func _callIt(body []byte) ([]byte, error) {
		var st MainStruct
		err := json.Unmarshal(body, &st)
		if err != nil {
			return nil, fmt.Errorf("Unmarshal() failed: %w", err)
		}
		`
	params := ""
	for _, prmNode := range ls.func_depends {
		VvarName := strings.ToUpper(prmNode.Name[0:1]) + prmNode.Name[1:] //1st letter must be upper
		params += fmt.Sprintf("&st.%s, ", VvarName)

	}
	params, _ = strings.CutSuffix(params, ", ")
	str += fmt.Sprintf("err = %s(%s)\n", ls.node.Name, params)

	str += `	if err != nil {
			return nil, fmt.Errorf("Chat() failed: %w", err)
		}
	
		res, err := json.Marshal(st)
		if err != nil {
			return nil, fmt.Errorf("Marshal() failed: %w", err)
		}
		return res, nil
	}`

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
					err := ls.addFuncDepend(prm.Names[0].Name)
					if err != nil {
						return err
					}
				}
			}
		}
	}

	return nil
}

func (ls *SANodeCode) updateMsgsDepends() error {

	//reset
	ls.msgs_depends = nil

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

				err := ls.addMsgDepend(nm)
				if err != nil {
					return err
				}
			}
			ln = ln[d2+1:]
		}
	}

	return nil
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

	//triggers
	for i, tr := range ls.Triggers {
		if tr == old_name {
			ls.Triggers[i] = new_name
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
