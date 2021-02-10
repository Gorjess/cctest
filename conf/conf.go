package conf

var (
	LenStackBuf = 4096

	// log
	LogLevel     string
	LogPath      string
	LogFileName  string
	LogChanNum   = 100000
	RollSize     uint32 //以M为单位
	EnableStdOut bool
	ErrToSkip    []string

	// oss
	OssUrl      string
	OssUsr      string
	OssPwd      string
	OssDB       string
	OssChanLen  uint32 = 10000
	OssInterval uint32 = 10

	// console
	ConsolePort   int
	ProfilePath   string
	ConsolePrompt = "Cloudcade# "
)
