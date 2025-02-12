package conf

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

type ServerCfg struct {
	MaxConnNum            int   `json:"max_conn_num"`
	MaxExecFuncTime       int   `json:"max_exec_func_time"`
	GatePendingWriteNum   int   `json:"gate_pending_write_num"`
	ConnNumPerSecond      int32 `json:"conn_num_per_second"`
	PlayerInteractiveTime int   `json:"player_interactive_time"`
}

var Server *ServerCfg

func Load() {
	Server = new(ServerCfg)
	cwd, _ := os.Getwd()
	println("cwd:", cwd)
	bs, e := ioutil.ReadFile("conf/config.json")
	if e != nil {
		panic(fmt.Sprintf("read gate config failed:%s", e.Error()))
	}

	e = json.Unmarshal(bs, Server)
	if e != nil {
		panic(fmt.Sprintf("unmarshal gate config failed:%s", e.Error()))
	}
}
