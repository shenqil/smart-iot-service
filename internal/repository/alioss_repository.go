package repository

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"rbac-admin/pkg/config"
	"strings"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

// OSSClient OSS客户端
type OSSClient struct {
	client *oss.Client
	bucket *oss.Bucket
	config *config.AliyunOSSConfig
}

// NewOSSClient 创建OSS客户端
func NewOSSClient() (*OSSClient, error) {

	conf := config.Conf.AliyunOSS

	// 创建OSSClient实例
	client, err := oss.New(conf.Endpoint, conf.AccessKeyId, conf.AccessKeySecret)
	if err != nil {
		return nil, fmt.Errorf("创建OSS客户端失败: %v", err)
	}

	// 获取存储空间
	bucket, err := client.Bucket(conf.BucketName)
	if err != nil {
		return nil, fmt.Errorf("获取存储空间失败: %v", err)
	}

	return &OSSClient{
		client: client,
		bucket: bucket,
		config: &conf,
	}, nil
}

// UploadFile 上传本地文件到OSS
func (o *OSSClient) UploadFile(localFilePath string, objectPath string) (string, error) {
	// 生成OSS上的文件路径
	objectName := o.generateObjectName(objectPath)

	// 上传文件
	err := o.bucket.PutObjectFromFile(objectName, localFilePath)
	if err != nil {
		return "", fmt.Errorf("上传文件失败: %v", err)
	}

	// 返回文件的访问URL
	return o.GetFileURL(objectName), nil
}

// UploadBytes 上传字节数据到OSS
func (o *OSSClient) UploadBytes(data []byte, objectPath string) (string, error) {
	// 生成OSS上的文件路径
	objectName := o.generateObjectName(objectPath)

	// 上传字节数据
	err := o.bucket.PutObject(objectName, bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("上传字节数据失败: %v", err)
	}

	return o.GetFileURL(objectName), nil
}

// UploadReader 从Reader上传文件到OSS
func (o *OSSClient) UploadReader(reader io.Reader, objectPath string) (string, error) {
	// 生成OSS上的文件路径
	objectName := o.generateObjectName(objectPath)

	// 上传数据
	err := o.bucket.PutObject(objectName, reader)
	if err != nil {
		return "", fmt.Errorf("上传Reader数据失败: %v", err)
	}

	return o.GetFileURL(objectName), nil
}

// UploadFromURL 从网络URL下载并上传到OSS
func (o *OSSClient) UploadFromURL(url string, objectPath string) (string, error) {
	// 创建自定义Transport跳过证书验证
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}

	// 下载文件
	resp, err := client.Get(url)
	if err != nil {
		return "", fmt.Errorf("下载文件失败: %v", err)
	}
	defer resp.Body.Close()

	// 检查HTTP状态码
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("下载失败，HTTP状态码: %d", resp.StatusCode)
	}

	// 如果没有指定对象路径，从URL中提取文件名
	if objectPath == "" {
		fileName := filepath.Base(url)
		// 移除查询参数
		if idx := strings.Index(fileName, "?"); idx != -1 {
			fileName = fileName[:idx]
		}
		objectPath = fileName
	}

	return o.UploadReader(resp.Body, objectPath)
}

// generateObjectName 生成OSS上的对象名称
func (o *OSSClient) generateObjectName(objectPath string) string {
	// 清理路径，去除前后的斜杠
	cleanPath := strings.Trim(objectPath, "/")

	// 组合上传目录和业务指定的路径
	if o.config.UploadDir != "" {
		uploadDir := strings.Trim(o.config.UploadDir, "/")
		return uploadDir + "/" + cleanPath
	}

	return cleanPath
}

// GetFileURL 获取文件的访问URL
func (o *OSSClient) GetFileURL(objectName string) string {

	// 返回普通URL
	return fmt.Sprintf("https://%s.%s/%s", o.config.BucketName, o.config.Endpoint, objectName)
}

// DeleteFile 删除OSS上的文件
func (o *OSSClient) DeleteFile(objectName string) error {
	err := o.bucket.DeleteObject(objectName)
	if err != nil {
		return fmt.Errorf("删除文件失败: %v", err)
	}
	return nil
}

// FileExists 检查文件是否存在
func (o *OSSClient) FileExists(objectName string) (bool, error) {
	return o.bucket.IsObjectExist(objectName)
}

// ListFiles 列出文件
func (o *OSSClient) ListFiles(prefix string) ([]oss.ObjectProperties, error) {
	var objects []oss.ObjectProperties

	marker := oss.Marker("")
	prefixOpt := oss.Prefix(prefix)
	for {
		lor, err := o.bucket.ListObjects(marker, prefixOpt)
		if err != nil {
			return nil, err
		}

		objects = append(objects, lor.Objects...)

		if !lor.IsTruncated {
			break
		}

		marker = oss.Marker(lor.NextMarker)
	}

	return objects, nil
}
