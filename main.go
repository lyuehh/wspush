package main

import (
	"bufio"
	ws "code.google.com/p/go.net/websocket"
	"fmt"
	"github.com/astaxie/beego/config"
	"github.com/ulricqin/goutils/filetool"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"time"
)

var cfgFile = "/etc/guacamole/guacamole.properties"
var Config config.ConfigContainer
var BaseLogDir string

func ReadLog(ws *ws.Conn) {
	var err error
	var dir string
	var file string

	for {

		var msg string

		if err = ws.Message.Receive(ws, &msg); err != nil {
			fmt.Println("Can't receive. Error:", err.Error())
			break
		}

		if msg == "" {
			ws.Message.Send(ws, "msg is blank")
			break
		}

		msg = strings.Trim(msg, " ")
		idx := strings.Index(msg, ";")
		if idx < 0 {
			ws.Message.Send(ws, "msg has no ';' ")
			break
		}

		dirStr := msg[0:idx]
		fileStr := msg[idx+1:]

		fmt.Println("dirStr:", dirStr, "; fileStr:", fileStr)

		dir = strings.Split(dirStr, ":")[1]
		if dir == "" {
			ws.Message.Send(ws, "dir is blank")
			break
		}

		file = strings.Split(fileStr, ":")[1]
		if file == "" {
			ws.Message.Send(ws, "file is blank")
			break
		}

		logPath := path.Join(BaseLogDir, dir, file)
		if !filetool.IsExist(logPath) {
			ws.Message.Send(ws, fmt.Sprintf("file: %s not exists", path.Join(dir, file)))
			break
		}

		var file *File

		file, err := os.Open(logPath)
		if err != nil {
			ws.Message.Send(ws, fmt.Sprintf("file: %s cannot open", path.Join(dir, file)))
			break
		}

		defer file.Close()

		reader := bufio.NewReader(file)
		var line []byte
		for {
			line, err = reader.ReadLine()
			if err == os.EOF {
				break
			} else {
				if err = ws.Message.Send(ws, string(line)); err != nil {
					fmt.Println("Can't send. Msg:", err.Error())
					break
				}
			}
			time.Sleep(5 * time.Millisecond)
		}
	}
}

func main() {
	fmt.Println("reading configuration file...")
	var err error
	Config, err = config.NewConfig("ini", "/etc/guacamole/guacamole.properties")
	if err != nil {
		fmt.Println("configuration file[/etc/guacamole/guacamole.properties] cannot parse.")
		os.Exit(1)
	}

	BaseLogDir = Config.String("message-storage")
	if BaseLogDir == "" {
		fmt.Println("no configuration item: message-storage")
		os.Exit(1)
	}

	http.Handle("/", http.FileServer(http.Dir("."))) // <-- note this line
	http.Handle("/ws", ws.Handler(ReadLog))

	if err := http.ListenAndServe(":8123", nil); err != nil {
		log.Fatal("ListenAndServe :8123 Error. Msg:", err)
	}

	fmt.Println("Http server start error")
}