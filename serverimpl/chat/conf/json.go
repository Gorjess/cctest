package conf

import (
	"cloudcadetest/framework/log"
	"encoding/json"
	"io/ioutil"
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
	bs, e := ioutil.ReadFile("./config.json")
	if e != nil {
		log.Error("read gate config failed:%s", e.Error())
		return
	}

	e = json.Unmarshal(bs, Server)
	if e != nil {
		log.Error("unmarshal gate config failed:%s", e.Error())
	}
}
