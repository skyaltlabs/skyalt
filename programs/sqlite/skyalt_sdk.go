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
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"strconv"
)

type SkyAltAttr struct {
	Name  string
	Value string

	Gui_type     string
	Gui_options  string
	Gui_ReadOnly bool //output
	Error        string
}

type SkyAltClient struct {
	Attrs []*SkyAltAttr
	conn  net.Conn
}

func (sa *SkyAltClient) FindAttr(name string) *SkyAltAttr {
	for _, attr := range sa.Attrs {
		if attr.Name == name {
			return attr
		}
	}
	return nil
}

func (sa *SkyAltClient) AddAttr(name, value string) *SkyAltAttr {
	attr := &SkyAltAttr{Name: name, Value: value}
	sa.Attrs = append(sa.Attrs, attr)
	return attr
}
func (sa *SkyAltClient) AddAttrOut(name, value string) *SkyAltAttr {
	attr := sa.AddAttr(name, value)
	attr.Gui_ReadOnly = true
	return attr
}

func (sa *SkyAltClient) recvPair() (string, string, bool) {

	var err error

	//size
	var t [8]byte
	_, err = sa.conn.Read(t[:]) //n? ...
	if err != nil {
		fmt.Printf("Read() failed: %v\n", err)
		return "", "", false
	}
	sz := binary.LittleEndian.Uint64(t[:])

	//data
	name := make([]byte, sz)
	_, err = sa.conn.Read(name) //n? ...
	if err != nil {
		fmt.Printf("Read() failed: %v\n", err)
		return "", "", false
	}

	//size
	_, err = sa.conn.Read(t[:]) //n? ...
	if err != nil {
		fmt.Printf("Read() failed: %v\n", err)
		return "", "", false
	}
	sz = binary.LittleEndian.Uint64(t[:])

	//data
	value := make([]byte, sz)
	_, err = sa.conn.Read(value) //n? ...
	if err != nil {
		fmt.Printf("Read() failed: %v\n", err)
		return "", "", false
	}

	return string(name), string(value), true
}

func (sa *SkyAltClient) recvPairNumber() (string, int, bool) {
	var value int

	name, val, ok := sa.recvPair()
	if ok {
		var err error
		value, err = strconv.Atoi(val)
		if err != nil {
			fmt.Printf("Atoi(%s) failed: %v\n", val, err)
			return "", 0, false
		}
	}
	return name, value, ok
}

func (sa *SkyAltClient) sendPair(name, value string) bool {

	var err error

	//size
	var t [8]byte
	binary.LittleEndian.PutUint64(t[:], uint64(len(name)))
	_, err = sa.conn.Write(t[:]) //n? ...
	if err != nil {
		fmt.Printf("Write() failed: %v\n", err)
		return false
	}

	//data
	_, err = sa.conn.Write([]byte(name)) //n? ...
	if err != nil {
		fmt.Printf("Write() failed: %v\n", err)
		return false
	}

	//size
	binary.LittleEndian.PutUint64(t[:], uint64(len(value)))
	_, err = sa.conn.Write(t[:]) //n? ...
	if err != nil {
		fmt.Printf("Write() failed: %v\n", err)
		return false
	}

	//data
	_, err = sa.conn.Write([]byte(value)) //n? ...
	if err != nil {
		fmt.Printf("Write() failed: %v\n", err)
		return false
	}

	return true
}

func (sa *SkyAltClient) sendPairNumber(name string, value int) bool {
	return sa.sendPair(name, strconv.Itoa(value))
}

func (sa *SkyAltClient) sendProgress(proc float64, Description string, err string) bool {

	if !sa.sendPairNumber("progress", 3) {
		return false
	}

	if !sa.sendPair("proc", strconv.FormatFloat(proc, 'f', -1, 64)) {
		return false
	}
	if !sa.sendPair("desc", Description) {
		return false
	}
	if !sa.sendPair("error", err) {
		return false
	}

	return true
}

func (sa *SkyAltClient) sendAttrs() bool {

	if !sa.sendPairNumber("attrs", len(sa.Attrs)) {
		return false
	}

	for _, attr := range sa.Attrs {

		//num pairs
		if !sa.sendPairNumber(attr.Name, 5) {
			return false
		}

		//values
		if !sa.sendPair("value", attr.Value) {
			return false
		}
		if !sa.sendPair("gui_type", attr.Gui_type) {
			return false
		}
		if !sa.sendPair("gui_options", attr.Gui_options) {
			return false
		}
		if !sa.sendPair("error", attr.Error) {
			return false
		}
		ro := "0"
		if attr.Gui_ReadOnly {
			ro = "1"
		}
		if !sa.sendPair("gui_read_only", ro) {
			return false
		}
	}

	return true
}

func (sa *SkyAltClient) Start(uid string, portStr string) {

	PORT, err := strconv.Atoi(portStr)
	if err != nil {
		fmt.Printf("Invalid PORT(%s) number: %v\n", portStr, err)
		os.Exit(1)
	}

	sa.conn, err = net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", PORT))
	if err != nil {
		fmt.Printf("Dial() failed: %v\n", err)
		os.Exit(1)
	}

	//send UID
	if !sa.sendPair("uid", uid) {
		os.Exit(1)
	}

	//send attributes
	if !sa.sendAttrs() {
		os.Exit(1)
	}
}

func (sa *SkyAltClient) Destroy() {
	err := sa.conn.Close()
	if err != nil {
		fmt.Printf("Close() failed: %v\n", err)
		os.Exit(1)
	}
}

func (sa *SkyAltClient) Get() bool {

	name, nAttrs, ok := sa.recvPairNumber()
	if !ok {
		os.Exit(1)
	}
	if name != "attrs" {
		fmt.Printf("Unknown message type(%s)\n", name)
		os.Exit(1)
	}

	for i := 0; i < nAttrs; i++ {
		name, nItems, ok := sa.recvPairNumber()
		if !ok {
			os.Exit(1)
		}
		attr := sa.FindAttr(name)
		if attr == nil {
			fmt.Printf("Warning: Attribute(%s) not found\n", name)
		}

		for ii := 0; ii < nItems; ii++ {
			name, value, ok := sa.recvPair()
			if !ok {
				os.Exit(1)
			}

			if attr == nil {
				continue
			}

			switch name {
			case "value":
				attr.Value = value
			case "gui_type":
				attr.Gui_type = value
			case "gui_options":
				attr.Gui_options = value
			case "error":
				attr.Error = value
			case "gui_read_only":
				val, err := strconv.Atoi(value)
				if err != nil {
					fmt.Printf("Atoi(%s) failed: %v\n", value, err)
					os.Exit(1)
				}
				attr.Gui_ReadOnly = (val != 0)

			default:
				fmt.Printf("Warning: Unknown attribute(%s)\n", name)
			}
		}
	}

	return true
}

func (sa *SkyAltClient) Progress(proc float64, Description string, sendOutputs bool) bool {
	if sendOutputs {
		if !sa.sendAttrs() {
			return false
		}
	}

	//progress
	if !sa.sendProgress(proc, Description, "") {
		return false
	}

	return true
}

func (sa *SkyAltClient) Finalize() bool {
	return sa.Progress(10, "", true)
}

func (sa *SkyAltClient) Error(errorStr string) bool {
	return sa.sendProgress(0, "", errorStr)
}
