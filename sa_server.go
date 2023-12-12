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
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"os/exec"
	"strconv"
	"sync/atomic"
)

const (
	SkyAltServer_READ_VALUES = 0

	SkyAltServer_WRITE_STRUCT   = 10
	SkyAltServer_WRITE_PROGRESS = 11
	SkyAltServer_WRITE_VALUES   = 12
)

type SkyAltServerValue struct {
	Name  string
	Value string

	Gui_type     string `json:",omitempty"`
	Gui_options  string `json:",omitempty"`
	Gui_ReadOnly bool   `json:",omitempty"` //output
}

type SkyAltServer struct {
	UID    string
	Values []*SkyAltServerValue
}

type ServerNodeProgress struct {
	Proc        float64
	Description string
	Error       string
}

type SANodeConn struct {
	strct    SkyAltServer
	progress ServerNodeProgress

	cmd  *exec.Cmd
	conn net.Conn
}

func (conn *SANodeConn) Destroy() {
	conn.conn.Close()
}

func (conn *SANodeConn) FindValue(name string) *SkyAltServerValue {
	for _, it := range conn.strct.Values {
		if it.Name == name {
			return it
		}
	}
	return nil
}

func (conn *SANodeConn) Start() bool {

	//Values
	js, err := json.Marshal(&conn.strct.Values)
	if err != nil {
		fmt.Printf("Unmarshal() Values failed: %v\n", err)
		return false
	}
	if !conn.write(js, SkyAltServer_READ_VALUES) {
		return false
	}

	conn.progress = ServerNodeProgress{} //reset

	for {
		js, tp, ok := conn.read()
		if !ok {
			return false
		}

		switch tp {
		case SkyAltServer_WRITE_PROGRESS:

			err := json.Unmarshal(js, &conn.progress)
			if err != nil {
				fmt.Printf("Unmarshal() failed: %v\n", err)
				return false
			}

			if conn.progress.Error != "" {
				return false
			}

			if conn.progress.Proc > 1.9 {
				return true //ok!
			}

		case SkyAltServer_WRITE_VALUES:
			err := json.Unmarshal(js, &conn.strct.Values)
			if err != nil {
				fmt.Printf("Unmarshal() failed: %v\n", err)
				return false
			}

		default:
			fmt.Printf("Unknown message type(%d)\n", tp)
			return false
		}
	}
}

func (conn *SANodeConn) write(js []byte, tp uint64) bool {

	//type
	var tt [8]byte
	binary.LittleEndian.PutUint64(tt[:], tp)
	_, err := conn.conn.Write(tt[:]) //n? ...
	if err != nil {
		fmt.Printf("Write() failed: %v\n", err)
		return false
	}

	//size
	var t [8]byte
	binary.LittleEndian.PutUint64(t[:], uint64(len(js)))
	_, err = conn.conn.Write(t[:]) //n? ...
	if err != nil {
		fmt.Printf("Write() failed: %v\n", err)
		return false
	}

	//data
	_, err = conn.conn.Write(js) //n? ...
	if err != nil {
		fmt.Printf("Write() failed: %v\n", err)
		return false
	}

	return true
}
func (conn *SANodeConn) read() ([]byte, uint64, bool) {

	//type
	var tp [8]byte
	_, err := conn.conn.Read(tp[:]) //n? ...
	if err != nil {
		fmt.Printf("Read() failed: %v\n", err)
		return nil, 0, false
	}

	//size
	var t [8]byte
	_, err = conn.conn.Read(t[:]) //n? ...
	if err != nil {
		fmt.Printf("Read() failed: %v\n", err)
		return nil, 0, false
	}
	sz := binary.LittleEndian.Uint64(t[:])

	//data
	js := make([]byte, sz)
	_, err = conn.conn.Read(js) //n? ...
	if err != nil {
		fmt.Printf("Read() failed: %v\n", err)
		return nil, 0, false
	}

	return js, binary.LittleEndian.Uint64(tp[:]), true
}

type SANodeServer struct {
	port int
	srv  net.Listener

	nodes_dir string
	nodes     []string

	interrupt atomic.Bool
}

func NewSANodeServer(nodes_dir string, port int) (*SANodeServer, error) {
	var server SANodeServer
	server.port = port
	server.nodes_dir = nodes_dir

	var err error
	server.srv, err = net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return nil, fmt.Errorf("Listen() failed: %w", err)
	}

	nodes := OsFileListBuild(nodes_dir, "", true)
	for _, n := range nodes.Subs {
		server.nodes = append(server.nodes, n.Name)
	}

	return &server, nil
}

func (server *SANodeServer) Destroy() {
	server.srv.Close()
}

func (server *SANodeServer) IsRunning() bool {
	return !server.interrupt.Load()
}

func (server *SANodeServer) Start(path string) *SANodeConn {

	uid := strconv.Itoa(rand.Int())

	cmd := exec.Command("./"+server.nodes_dir+"/"+path+"/main", uid, strconv.Itoa(server.port))
	cmd.Start()

	c, err := server.srv.Accept()
	if err != nil {
		fmt.Printf("Accept() failed: %v\n", err)
		return nil
	}

	node := SANodeConn{conn: c, cmd: cmd}
	js, tp, ok := node.read()
	if !ok {
		return nil
	}

	if tp != SkyAltServer_WRITE_STRUCT {
		fmt.Printf("type(%d) != SkyAltServer_WRITE_STRUCT\n", tp)
		return nil
	}

	err = json.Unmarshal(js, &node.strct)
	if err != nil {
		fmt.Printf("Unmarshal() failed: %v\n", err)
		return nil
	}

	if node.strct.UID != uid {
		fmt.Printf("uid(%s) != uid(%s)\n", node.strct.UID, uid)
		return nil
	}

	//cmd.Wait()
	//cmd.Cancel()

	return &node
}
