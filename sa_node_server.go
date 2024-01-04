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
	"errors"
	"fmt"
	"math/rand"
	"net"
	"os/exec"
	"strconv"
	"sync"
)

type SkyAltServerAttr struct {
	Name  string
	Value string
	Error string
}

type SANodeConn struct {
	program string
	Attrs   []*SkyAltServerAttr

	cmd  *exec.Cmd
	conn net.Conn

	lock sync.Mutex
}

func (conn *SANodeConn) Destroy() {
	conn.conn.Close()
}

func (conn *SANodeConn) Lock() {
	conn.lock.Lock()
}
func (conn *SANodeConn) Unlock() {
	conn.lock.Unlock()
}

func (conn *SANodeConn) FindAttr(name string) *SkyAltServerAttr {
	for _, it := range conn.Attrs {
		if it.Name == name {
			return it
		}
	}
	return nil
}

func (conn *SANodeConn) AddAttr(name string) *SkyAltServerAttr {

	attr := conn.FindAttr(name)
	if attr == nil {
		attr = &SkyAltServerAttr{Name: name}
		conn.Attrs = append(conn.Attrs, attr)
	}
	return attr
}

func (conn *SANodeConn) sendAttrs() bool {

	if !conn.sendPairNumber("attrs", len(conn.Attrs)) {
		return false
	}

	for _, attr := range conn.Attrs {

		//num pairs
		if !conn.sendPairNumber(attr.Name, 5) {
			return false
		}

		//values
		if !conn.sendPair("value", attr.Value) {
			return false
		}
		if !conn.sendPair("error", attr.Error) {
			return false
		}
	}

	return true
}

func (conn *SANodeConn) recvAttrs(numAttrs int) bool {

	for i := 0; i < numAttrs; i++ {
		name, num2, ok := conn.recvPairNumber()
		if !ok {
			return false
		}
		attr := conn.AddAttr(name)

		for ii := 0; ii < num2; ii++ {
			name, value, ok := conn.recvPair()
			if !ok {
				return false
			}

			switch name {
			case "value":
				attr.Value = value
			case "error":
				attr.Error = value
			default:
				fmt.Printf("Warning: Unknown name(%s)\n", name)
			}
		}
	}
	return true
}

func (conn *SANodeConn) Run(node *SANode) bool {

	//attributes
	conn.sendAttrs()

	//reset
	//conn.progress = ServerNodeProgress{}

	for {
		name, num, ok := conn.recvPairNumber()
		if !ok {
			return false
		}

		switch name {
		case "progress":
			for i := 0; i < num; i++ {
				name, value, ok := conn.recvPair()
				if !ok {
					return false
				}
				switch name {
				case "proc":
					node.progress, _ = strconv.ParseFloat(value, 64)
				case "desc":
					node.progress_desc = value
				case "error":
					if value != "" {
						node.errExe = errors.New(value)
					} else {
						node.errExe = nil
					}
				default:
					fmt.Printf("Warning: Unknown name(%s)\n", name)
				}
			}

			if node.errExe != nil {
				return false
			}

			if node.progress > 1.9 {
				return true //ok!
			}

		case "attrs":
			if !conn.recvAttrs(num) {
				return false
			}

		default:
			fmt.Printf("Unknown message name(%s)\n", name)
			return false
		}
	}
}

func (conn *SANodeConn) _read(buff []byte) error {
	p := 0
	for p < len(buff) {
		n, err := conn.conn.Read(buff[p:])
		if err != nil {
			return err
		}
		p += n
	}

	return nil
}
func (conn *SANodeConn) _write(buff []byte) error {
	p := 0
	for p < len(buff) {
		n, err := conn.conn.Write(buff[p:])
		if err != nil {
			return err
		}
		p += n
	}

	return nil
}

func (conn *SANodeConn) recvPair() (string, string, bool) {

	var err error

	//size
	var t [8]byte
	err = conn._read(t[:])
	if err != nil {
		fmt.Printf("Read() failed: %v\n", err)
		return "", "", false
	}
	sz := binary.LittleEndian.Uint64(t[:])

	//data
	name := make([]byte, sz)
	err = conn._read(name)
	if err != nil {
		fmt.Printf("Read() failed: %v\n", err)
		return "", "", false
	}

	//size
	err = conn._read(t[:])
	if err != nil {
		fmt.Printf("Read() failed: %v\n", err)
		return "", "", false
	}
	sz = binary.LittleEndian.Uint64(t[:])

	//data
	value := make([]byte, sz)
	err = conn._read(value)
	if err != nil {
		fmt.Printf("Read() failed: %v\n", err)
		return "", "", false
	}

	return string(name), string(value), true
}

func (conn *SANodeConn) recvPairNumber() (string, int, bool) {
	var value int

	name, val, ok := conn.recvPair()
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

func (conn *SANodeConn) sendPair(name, value string) bool {

	var err error

	//size
	var t [8]byte
	binary.LittleEndian.PutUint64(t[:], uint64(len(name)))
	err = conn._write(t[:])
	if err != nil {
		fmt.Printf("Write() failed: %v\n", err)
		return false
	}

	//data
	err = conn._write([]byte(name))
	if err != nil {
		fmt.Printf("Write() failed: %v\n", err)
		return false
	}

	//size
	binary.LittleEndian.PutUint64(t[:], uint64(len(value)))
	err = conn._write(t[:])
	if err != nil {
		fmt.Printf("Write() failed: %v\n", err)
		return false
	}

	//data
	err = conn._write([]byte(value))
	if err != nil {
		fmt.Printf("Write() failed: %v\n", err)
		return false
	}

	return true
}

func (conn *SANodeConn) sendPairNumber(name string, value int) bool {
	return conn.sendPair(name, strconv.Itoa(value))
}

type SANodeServer struct {
	port int
	srv  net.Listener

	nodes_dir string
	nodes     []string

	connections []*SANodeConn
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
	//close connections
	for _, it := range server.connections {
		it.Destroy()
	}
	server.connections = nil

	//close server
	server.srv.Close()
	server.srv = nil
}

func (server *SANodeServer) Interrupt() {
	for _, it := range server.connections {
		it.Destroy()
	}
	server.connections = nil
}

func (server *SANodeServer) Start(program string) *SANodeConn {

	//find
	for _, it := range server.connections {
		if it.program == program {
			return it
		}
	}

	//run
	uid := strconv.Itoa(rand.Int())

	exePath := server.nodes_dir + "/" + program + "/main"
	if !OsFileExists(exePath) {
		fmt.Printf("Program(%s) not exist\n", exePath)
		return nil
	}

	var conn *SANodeConn

	//lock/unlock? .....
	{
		cmd := exec.Command("./"+exePath, uid, strconv.Itoa(server.port))
		cmd.Start()

		c, err := server.srv.Accept() //multi-threading ..... + co deadlock(no link) .....
		if err != nil {
			fmt.Printf("Accept() from program(%s) failed: %v\n", exePath, err)
			return nil
		}

		conn = &SANodeConn{program: program, conn: c, cmd: cmd}
	}

	//UID
	name, value, ok := conn.recvPair()
	if !ok {
		return nil
	}
	if name != "uid" {
		fmt.Printf("name(%s) != 'uid'\n", name)
		return nil
	}

	if value != uid {
		fmt.Printf("recv uid(%s) != uid(%s)\n", value, uid)
		return nil
	}

	//Attributes
	name, numAttrs, ok := conn.recvPairNumber()
	if !ok {
		return nil
	}
	if name != "attrs" {
		fmt.Printf("name(%s) != 'attrs'\n", name)
		return nil
	}
	if !conn.recvAttrs(numAttrs) {
		return nil
	}

	//cmd.Wait()
	//cmd.Cancel()

	server.connections = append(server.connections, conn)

	return conn
}
