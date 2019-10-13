package server

import (
	"encoding/json"
	"errors"
	"fmt"
	_ "github.com/gin-gonic/gin"
	"go-leaf/core"
	"net/http"
	"strconv"
)

type allocResponse struct {
	Errno int    `json:"errno"`
	Msg   string `json:"msg"`
	Id    int64  `json:"id"`
}

func httpsegmentGetid(w http.ResponseWriter, r *http.Request) {
	var (
		resp   allocResponse = allocResponse{}
		err    error
		bytes  []byte
		bizTag string
	)

	if err = r.ParseForm(); err != nil {
		goto RESP
	}
	//获取标签名
	if bizTag = r.Form.Get("biz_tag"); bizTag == "" {
		err = errors.New("need biz_tag param")
		goto RESP
	}

	for { // 跳过ID=0, 一般业务不支持ID=0
		if resp.Id, err = core.GAllocSegment.NextId(bizTag); err != nil {
			goto RESP
		}
		if resp.Id != 0 {
			break
		}
	}

RESP:
	if err != nil {
		resp.Errno = -1
		resp.Msg = fmt.Sprintf("%v", err)
		w.WriteHeader(500)
	} else {
		resp.Msg = "success"
	} //->json
	if bytes, err = json.Marshal(&resp); err == nil {
		w.Write(bytes)
	} else {
		w.WriteHeader(500)
	}
}

func httpsnowflakeGetid(w http.ResponseWriter, r *http.Request) {
	var (
		resp      allocResponse = allocResponse{}
		err       error
		bytes     []byte
		machineId int
		str       string
	)
	if err = r.ParseForm(); err != nil {
		goto RESP
	}
	//获取标签名
	if str = r.Form.Get("machineid"); str == "" {
		err = errors.New("need machineid param")
		goto RESP
	}
	if machineId, err = strconv.Atoi(str); err != nil {
		goto RESP
	}
	if exist := core.GAllocSnowflake.BoolmachineMap(int32(machineId)); exist {
		goto RESP
	} else {
		if err = core.GAllocSnowflake.NewSnowflake(int32(machineId)); err != nil {
			goto RESP
		}
	}
	//snow,_ := core.GAllocSnowflake.MachineMap[uint16(machineId)]
	//if resp.Id,err =snow.NextId(); err != nil{
	//	goto RESP
	//}

RESP:
	snow, _ := core.GAllocSnowflake.MachineMap[int32(machineId)]
	resp.Id, err = snow.NextId()

	if err != nil {
		resp.Errno = -1
		resp.Msg = fmt.Sprintf("%v", err)
		w.WriteHeader(500)
	} else {
		resp.Msg = "success"
	} //->json

	if bytes, err = json.Marshal(&resp); err == nil {
		w.Write(bytes)
	} else {
		w.WriteHeader(500)
	}
}
func Inithttp() {
	//http://localhost:8000/alloc-snowflake?machineid=11
	http.HandleFunc("/alloc-snowflake", httpsnowflakeGetid)
	//http://localhost:8000/alloc-segment?biz_tag=test
	http.HandleFunc("/alloc-segment", httpsegmentGetid)

	http.ListenAndServe(":8000", nil)
}

//func ginGetid(c *gin.Context) {
//	c.String(http.StatusOK,"hello world")
//}
//func Initgin(){
//	r := gin.Default()
//	r.GET("/alloc",ginGetid )
//	r.Run(":8010")
//}
