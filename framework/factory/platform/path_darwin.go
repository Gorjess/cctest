package platform

func GetRootPath() string {
	return "/tmp/service/"
}

func GetLogRootPath() string {
	return GetRootPath() + "/log"
}
