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
	"fmt"
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

type SANodeCode struct {
	node *SANode

	Triggers    []string //nodes
	TempCommand string
	Command     string
	Answer      string

	depends []*SANode
	prompt  string
	code    string

	exeTimeSec float64

	//prompt history? ...
}

func InitSANodeCode(node *SANode) SANodeCode {
	ls := SANodeCode{}
	ls.node = node
	return ls
}

func (ls *SANodeCode) updateLinks(node *SANode) error {
	ls.node = node

	if !node.IsCode() {
		return nil
	}

	//refresh
	err := ls.buildPrompt()
	if err != nil {
		return err
	}
	err = ls.buildCode()
	if err != nil {
		return err
	}

	filePath := "temp/go/" + ls.node.Name + ".go"

	fl, err := os.ReadFile(filePath)
	if err != nil || !bytes.Equal(fl, []byte(ls.code)) {

		//write
		{
			OsFolderCreate("temp/go/")
			err = os.WriteFile(filePath, []byte(ls.code), 0644)
			if err != nil {
				return err
			}
		}
		//compile
		{
			cmd := exec.Command("go", "build", ls.node.Name+".go")
			cmd.Dir = "temp/go/"
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			err = cmd.Start()
			if err != nil {
				return err
			}
			cmd.Wait()
		}

	}

	return nil
}

func IsWordLetter(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch == '_'
}

func (ls *SANodeCode) renameNode(old_name string, new_name string) error {

	//command
	ls.TempCommand = strings.ReplaceAll(ls.TempCommand, "'"+old_name+"'", "'"+new_name+"'")
	ls.Command = strings.ReplaceAll(ls.Command, "'"+old_name+"'", "'"+new_name+"'")

	//answer
	act := 0
	for {
		d := strings.Index(ls.Answer[act:], old_name)
		if d >= 0 {
			st := act + d
			en := act + d + len(old_name)

			isValid := true
			if st > 0 && IsWordLetter(ls.Answer[st-1]) {
				isValid = false
			}
			if en < len(ls.Answer) && IsWordLetter(ls.Answer[en]) {
				isValid = false
			}

			if isValid {
				ls.Answer = ls.Answer[:st] + new_name + ls.Answer[en:]
				act = act + d + len(new_name)
			} else {
				act = act + d + len(old_name)
			}
		} else {
			break
		}
	}

	//triggers
	for i, tr := range ls.Triggers {
		if tr == old_name {
			ls.Triggers[i] = new_name
		}
	}

	//refresh
	err := ls.updateLinks(ls.node)
	if err != nil {
		return err
	}

	return nil
}

func (ls *SANodeCode) addTrigger(name string) {
	for _, tr := range ls.Triggers {
		if tr == name {
			return
		}
	}
	ls.Triggers = append(ls.Triggers, name)
}

func (ls *SANodeCode) addDepend(node *SANode) {
	//find
	for _, dp := range ls.depends {
		if dp.Name == node.Name {
			return //already in
		}
	}

	//add
	ls.depends = append(ls.depends, node)
}

func (ls *SANodeCode) extractDepends() error {

	cmd := ls.Command
	ls.depends = nil

	for {
		d1 := strings.IndexByte(cmd, '\'')
		if d1 >= 0 {
			cmd = cmd[d1+1:]
		} else {
			break
		}
		d2 := strings.IndexByte(cmd, '\'')
		if d2 >= 0 {
			nm := cmd[:d2]
			node := ls.node.FindNode(nm)
			if node != nil {
				ls.addDepend(node)
			} else {
				fmt.Printf("Warning: Node(%s) not found\n", nm)
			}
			cmd = cmd[d2+1:]
		} else {
			return fmt.Errorf("2nd ' not found")
		}
	}

	return nil
}

func (ls *SANodeCode) extractCode() (string, error) {
	code := ls.Answer

	d := strings.Index(code, "```go")
	if d >= 0 {
		code = code[d+5:]

		d = strings.Index(code, "```")
		if d >= 0 {
			code = code[:d]
		} else {
			return "", fmt.Errorf("code_end not found")
		}
	} else {
		return "", fmt.Errorf("no code in answer")
	}

	return code, nil
}

func (ls *SANodeCode) extractImports(code string) ([]string, error) {
	var imports []string

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, "test.go", "package main\n\n"+code, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("ParseFile() failed: %w", err)
	}

	for _, im := range node.Imports {
		imports = append(imports, im.Path.Value)
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

func (ls *SANodeCode) buildSqlInfos() (string, error) {
	str := ""

	for _, prmNode := range ls.depends {
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

func (ls *SANodeCode) buildPrompt() error {

	ls.prompt = ""

	ls.extractDepends()

	str := "I have this golang code:\n\n"
	str += g_code_const_gpt + "\n"

	params := ""
	for _, prmNode := range ls.depends {

		StructName := strings.ToUpper(prmNode.Exe[0:1]) + prmNode.Exe[1:] //1st letter must be upper
		params += fmt.Sprintf("%s *%s, ", prmNode.Name, StructName)
	}
	params, _ = strings.CutSuffix(params, ", ")
	str += fmt.Sprintf("\nfunc %s(%s) error {\n\n}\n\n", ls.node.Name, params)

	str += fmt.Sprintf("You can change the code only inside '%s' function and output only import(s) and '%s' function code. Don't explain the code.\n", ls.node.Name, ls.node.Name)

	strSQL, err := ls.buildSqlInfos()
	if err != nil {
		return err
	}
	str += strSQL

	str += "Your job: " + ls.Command

	ls.prompt = str
	return nil
}

func (ls *SANodeCode) buildCode() error {

	code, err := ls.extractCode()
	if err != nil {
		return err
	}
	imports, err := ls.extractImports(code)
	if err != nil {
		return err
	}
	fn, err := ls.extractFunc(code)
	if err != nil {
		return err
	}

	str := "package main\n\n"

	//imports
	for _, imp := range g_str_imports {
		str += fmt.Sprintf("import %s\n", imp)
	}
	for _, imp := range imports {
		found := false
		for _, imp2 := range g_str_imports {
			if imp == imp2 {
				found = true
				break
			}
		}
		if !found {
			str += fmt.Sprintf("import %s\n", imp)
		}
	}
	str += "\n"

	//struct
	str += "type MainStruct struct {\n"
	for _, prmNode := range ls.depends {
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
	for _, prmNode := range ls.depends {
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

	ls.code = str
	return nil
}

func (ls *SANodeCode) GetAnswer() error {
	oldCommand := ls.Command

	ls.Command = ls.TempCommand

	err := ls.buildPrompt()
	if err != nil {
		return err
	}

	messages := fmt.Sprintf(`[{"role": "system", "content": "You are ChatGPT, an AI assistant. Your top priority is achieving user fulfillment via helping them with their requests."}, {"role": "user", "content": %s}]`, OsText_RAWtoJSON(ls.prompt))
	g4f := ls.node.app.base.services.GetG4F()
	answer, err := g4f.Complete(&SAServiceG4FProps{Model: "gpt-4-turbo", Messages: messages})
	if err != nil {
		ls.Command = oldCommand
		return fmt.Errorf("Complete() failed: %w", err)
	}
	ls.Answer = string(answer)

	//refresh
	err = ls.updateLinks(ls.node)
	if err != nil {
		ls.Command = oldCommand
		return err
	}

	return nil
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

func (ls *SANodeCode) Execute() error {
	st := OsTime()

	ls.node.errExe = nil //reset

	//input
	{
		vars := make(map[string]interface{})
		for _, prmNode := range ls.depends {
			vars[prmNode.Name] = prmNode.Attrs

			attrs := prmNode.Attrs
			if prmNode.HasNodeAttr() {
				attrs = make(map[string]interface{})
				attrs["node"] = prmNode.Name

			}
			vars[prmNode.Name] = attrs
		}
		inputJs, err := json.Marshal(vars)
		if err != nil {
			return fmt.Errorf("Marshal() failed: %w", err)
		}

		ls.node.app.base.services.SetJob(inputJs, ls.node.app)
	}

	//process
	{
		cmd := exec.Command("./temp/go/"+ls.node.Name, strconv.Itoa(ls.node.app.base.services.port))
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Start()
		if err != nil {
			return err
		}
		defer cmd.Wait()
	}

	//output
	{
		outputJs := ls.node.app.base.services.GetResult()

		var vars map[string]interface{}
		err := json.Unmarshal(outputJs, &vars)
		if err != nil {
			return fmt.Errorf("Unmarshal() failed: %w", err)
		}

		for key, node := range vars {
			prmNode := ls.node.FindNode(key)
			if prmNode != nil {
				if !prmNode.HasNodeAttr() {
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

	return nil
}
