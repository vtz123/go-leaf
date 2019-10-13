package main

import (
	"go-leaf/core"
	"go-leaf/server"
	"log"
	_ "log"
)

func main() {

	if err := core.Init(); err != nil {
		log.Print(err)
	}

	//开启http服务
	server.Inithttp() //:8000

}
