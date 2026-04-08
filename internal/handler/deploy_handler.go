package handler

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"rbac-admin/internal/handler/response"
	"rbac-admin/pkg/config"
	"rbac-admin/pkg/constant"
	"rbac-admin/pkg/contextx"
	"rbac-admin/pkg/errx"
	"strings"

	"github.com/gin-gonic/gin"
)

// DeployHandler 部署接口处理器
type DeployHandler struct {
}

// Deploy 部署 dist 压缩包
// @Summary 部署 dist 压缩包
// @Description 上传 dist 压缩包，解压到部署目录
// @Tags 部署管理
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "压缩包文件（支持 zip, tar.gz）"
// @Success 200 {object} dto.Response "部署成功"
// @Failure 400 {object} dto.Response "请求参数错误"
// @Failure 500 {object} dto.Response "内部服务器错误"
// @Router /api/v1/admin-auth/deploy [post]
func (h *DeployHandler) Deploy(c *gin.Context) {
	logger := contextx.GetLoggerFromContext(c.Request.Context())

	// 获取上传的文件
	file, err := c.FormFile("file")
	if err != nil {
		logger.Error(fmt.Sprintf("获取上传文件失败：%v", err))
		response.Error(c, errx.New(constant.BadRequest, "请选择要上传的文件", err))
		return
	}

	// 检查文件扩展名
	fileName := file.Filename
	fileNameLower := strings.ToLower(fileName)
	isTarGz := strings.HasSuffix(fileNameLower, ".tar.gz")
	isZip := strings.HasSuffix(fileNameLower, ".zip")
	
	if !isZip && !isTarGz {
		logger.Error(fmt.Sprintf("不支持的文件格式：%s，只支持 .zip 和 .tar.gz 格式", fileName))
		response.Error(c, errx.New(constant.BadRequest, "只支持 zip 和 tar.gz 格式的压缩包", nil))
		return
	}

	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "deploy-*")
	if err != nil {
		logger.Error(fmt.Sprintf("创建临时目录失败：%v", err))
		response.Error(c, errx.New(constant.InternalServerError, "创建临时目录失败", err))
		return
	}
	defer os.RemoveAll(tmpDir) // 清理临时目录

	// 保存上传的文件到临时目录
	tmpFilePath := filepath.Join(tmpDir, fileName)
	if err := c.SaveUploadedFile(file, tmpFilePath); err != nil {
		logger.Error(fmt.Sprintf("保存上传文件失败：%v", err))
		response.Error(c, errx.New(constant.InternalServerError, "保存上传文件失败", err))
		return
	}

	// 获取部署目录
	deployDir := config.Conf.Deploy.DeployDir
	if deployDir == "" {
		deployDir = "./dist"
	}

	// 确保部署目录存在
	if err := os.MkdirAll(deployDir, 0755); err != nil {
		logger.Error(fmt.Sprintf("创建部署目录失败：%v", err))
		response.Error(c, errx.New(constant.InternalServerError, "创建部署目录失败", err))
		return
	}

	// 删除部署目录下的所有文件
	if err := clearDirectory(deployDir); err != nil {
		logger.Error(fmt.Sprintf("清空部署目录失败：%v", err))
		response.Error(c, errx.New(constant.InternalServerError, "清空部署目录失败", err))
		return
	}

	// 解压文件
	if isTarGz {
		if err := extractTarGz(tmpFilePath, deployDir); err != nil {
			logger.Error(fmt.Sprintf("解压 tar.gz 文件失败：%v", err))
			response.Error(c, errx.New(constant.InternalServerError, "解压文件失败", err))
			return
		}
	} else if isZip {
		if err := extractZip(tmpFilePath, deployDir); err != nil {
			logger.Error(fmt.Sprintf("解压 zip 文件失败：%v", err))
			response.Error(c, errx.New(constant.InternalServerError, "解压文件失败", err))
			return
		}
	}

	response.Success(c, gin.H{
		"message":    "部署成功",
		"deploy_dir": deployDir,
	})
}

// clearDirectory 清空目录下的所有文件和子目录
func clearDirectory(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		path := filepath.Join(dir, entry.Name())
		if err := os.RemoveAll(path); err != nil {
			return err
		}
	}

	return nil
}

// extractZip 解压 zip 文件
func extractZip(zipPath, destDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		// 防止路径遍历攻击
		path := filepath.Join(destDir, f.Name)
		if !strings.HasPrefix(path, filepath.Clean(destDir)+string(os.PathSeparator)) {
			return fmt.Errorf("非法的文件路径：%s", f.Name)
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(path, f.FileInfo().Mode())
			continue
		}

		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			return err
		}

		outFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.FileInfo().Mode())
		if err != nil {
			rc.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()

		if err != nil {
			return err
		}
	}

	return nil
}

// extractTarGz 解压 tar.gz 文件
func extractTarGz(tarGzPath, destDir string) error {
	file, err := os.Open(tarGzPath)
	if err != nil {
		return err
	}
	defer file.Close()

	gzr, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// 防止路径遍历攻击
		path := filepath.Join(destDir, header.Name)
		if !strings.HasPrefix(path, filepath.Clean(destDir)+string(os.PathSeparator)) {
			return fmt.Errorf("非法的文件路径：%s", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(path, os.FileMode(header.Mode)); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
				return err
			}

			outFile, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return err
			}

			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return err
			}
			outFile.Close()
		}
	}

	return nil
}

