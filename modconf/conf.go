package modconf

type ServerConf struct {
	LenStackBuf  int    `json:"len_stack_buf"`
	LogLevel     string `json:"log_level"`
	LogPath      string `json:"log_path"`
	LogFileName  string `json:"log_file_name"`
	LogChanNum   int    `json:"log_chan_num"`
	RollSize     uint32 `json:"roll_size"` // MB
	EnableStdOut bool   `json:"enable_std_out"`
}
