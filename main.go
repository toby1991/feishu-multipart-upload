package main

import (
	"fmt"
	"github.com/toby1991/feishu-multipart-upload/upload"
	"os"
)

func main() {
	// 检查命令行参数
	if len(os.Args) != 5 {
		fmt.Println("使用方法: ./程序名 YOUR_APP_ID YOUR_APP_SECRET /xxx/xxx/abc.wav FEISHU_PATH")
		os.Exit(1)
	}

	// 从命令行参数获取值
	appID := os.Args[1]
	appSecret := os.Args[2]
	filePath := os.Args[3]
	parentNode := os.Args[4]

	if err := upload.Upload(appID, appSecret, filePath, parentNode); err != nil {
		fmt.Println(err)
	}
}
