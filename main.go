package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"

	lark "github.com/larksuite/oapi-sdk-go/v3"
	larkdrive "github.com/larksuite/oapi-sdk-go/v3/service/drive/v1"
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

	// 创建 Client
	client := lark.NewClient(appID, appSecret)

	// 获取文件信息
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Printf("打开文件失败: %v\n", err)
		return
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		fmt.Printf("获取文件信息失败: %v\n", err)
		return
	}

	// 步骤1: 准备上传
	uploadInfo, err := prepareUpload(client, fileInfo, parentNode)
	if err != nil {
		fmt.Printf("准备上传失败: %v\n", err)
		return
	}
	fmt.Println(uploadInfo)

	// 步骤2: 分片上传
	err = uploadParts(client, file, uploadInfo)
	if err != nil {
		fmt.Printf("分片上传失败: %v\n", err)
		return
	}

	// 步骤3: 完成上传
	fileToken, err := finishUpload(client, uploadInfo)
	if err != nil {
		fmt.Printf("完成上传失败: %v\n", err)
		return
	}

	fmt.Printf("文件上传成功，文件Token: %s\n", fileToken)
}

type UploadInfo struct {
	UploadID  string
	BlockSize int
	BlockNum  int
}

func prepareUpload(client *lark.Client, fileInfo os.FileInfo, parentNode string) (*UploadInfo, error) {
	req := larkdrive.NewUploadPrepareFileReqBuilder().
		FileUploadInfo(larkdrive.NewFileUploadInfoBuilder().
			FileName(filepath.Base(fileInfo.Name())).
			ParentType("explorer").
			ParentNode(parentNode). // 替换为实际的父节点ID
			Size(int(fileInfo.Size())).
			Build()).
		Build()

	resp, err := client.Drive.File.UploadPrepare(context.Background(), req)
	if err != nil {
		return nil, err
	}

	if !resp.Success() {
		return nil, fmt.Errorf("准备上传失败: %s", resp.Msg)
	}

	blockNum := int(math.Ceil(float64(fileInfo.Size()) / float64(*resp.Data.BlockSize)))
	return &UploadInfo{
		UploadID:  *resp.Data.UploadId,
		BlockSize: *resp.Data.BlockSize,
		BlockNum:  blockNum,
	}, nil
}
func uploadParts(client *lark.Client, file *os.File, uploadInfo *UploadInfo) error {
	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("获取文件信息失败: %v", err)
	}
	totalSize := fileInfo.Size()
	remainingSize := int(totalSize)

	for i := int(0); i < uploadInfo.BlockNum; i++ {
		var partSize int
		if remainingSize > uploadInfo.BlockSize {
			partSize = uploadInfo.BlockSize
		} else {
			partSize = remainingSize
		}

		if partSize == 0 {
			break // 不上传空分片
		}

		buffer := make([]byte, partSize)
		n, err := io.ReadFull(file, buffer)
		if err != nil && err != io.EOF && !errors.Is(err, io.ErrUnexpectedEOF) {
			return fmt.Errorf("读取文件分片失败: %v", err)
		}

		req := larkdrive.NewUploadPartFileReqBuilder().
			Body(larkdrive.NewUploadPartFileReqBodyBuilder().
				UploadId(uploadInfo.UploadID).
				Seq(i).
				Size(int(n)).
				File(bytes.NewReader(buffer[:n])).
				Build()).
			Build()

		resp, err := client.Drive.File.UploadPart(context.Background(), req)
		if err != nil {
			return fmt.Errorf("上传分片 %d 失败: %v", i+1, err)
		}

		if !resp.Success() {
			return fmt.Errorf("上传分片 %d 失败: %s", i+1, resp.Msg)
		}

		fmt.Printf("上传分片 %d 成功，大小：%d 字节\n", i+1, n)
		remainingSize -= int(n)
	}

	return nil
}

func finishUpload(client *lark.Client, uploadInfo *UploadInfo) (string, error) {
	req := larkdrive.NewUploadFinishFileReqBuilder().
		Body(larkdrive.NewUploadFinishFileReqBodyBuilder().
			UploadId(uploadInfo.UploadID).
			BlockNum(uploadInfo.BlockNum).
			Build()).
		Build()

	resp, err := client.Drive.File.UploadFinish(context.Background(), req)
	if err != nil {
		return "", err
	}

	if !resp.Success() {
		return "", fmt.Errorf("完成上传失败: %s", resp.Msg)
	}

	return *resp.Data.FileToken, nil
}
