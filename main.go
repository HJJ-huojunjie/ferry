package main

import (
	"ferry/cmd"
	"time"
)

func init() {
	// 强制运行时本地时区为东八区，避免容器默认 UTC 导致
	// time.Now().Format(...) 以及 GORM loc=Local 解析出现 8 小时偏差。
	if loc, err := time.LoadLocation("Asia/Shanghai"); err == nil {
		time.Local = loc
	} else {
		time.Local = time.FixedZone("CST", 8*3600)
	}
}

func main() {
	cmd.Execute()
}
