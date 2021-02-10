package platform

func GetRootPath() string {
	return "D:\\service"
}

func GetLogRootPath() string {
	return GetRootPath() + "/log"
}
