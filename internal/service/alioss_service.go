package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"rbac-admin/internal/model/dto"
	"rbac-admin/pkg/config"
	"rbac-admin/pkg/constant"
	"rbac-admin/pkg/contextx"
	"rbac-admin/pkg/errx"
	"time"
)

// AliyunOSSService OSS服务
type AliyunOSSService struct {
}

type ConfigStruct struct {
	Expiration string     `json:"expiration"`
	Conditions [][]string `json:"conditions"`
}

func getGMTISO8601(expireEnd int64) string {
	return time.Unix(expireEnd, 0).UTC().Format("2006-01-02T15:04:05Z")
}

// GetPolicyToken  OSS 上传所需的签名和凭证
func (s *AliyunOSSService) GetPolicyToken(ctx context.Context, req *dto.PolicyTokenRequest) (*dto.PolicyTokenResponse, error) {

	logger := contextx.GetLoggerFromContext(ctx)
	confAliyunOSS := config.Conf.AliyunOSS

	uploadDir := confAliyunOSS.UploadDir + req.SubDir
	host := fmt.Sprintf("https://%s.%s", confAliyunOSS.BucketName, confAliyunOSS.Endpoint)

	now := time.Now().Unix()
	expireEnd := now + confAliyunOSS.ExpireTime
	tokenExpire := getGMTISO8601(expireEnd)
	var config ConfigStruct
	config.Expiration = tokenExpire
	var condition []string
	condition = append(condition, "starts-with")
	condition = append(condition, "$key")
	condition = append(condition, uploadDir)
	config.Conditions = append(config.Conditions, condition)
	result, err := json.Marshal(config)
	if err != nil {
		logger.Error("callback json err", slog.Any("error", err))
		return nil, errx.New(constant.InternalServerError, constant.MsgInternalServerErr, err)
	}
	encodedResult := base64.StdEncoding.EncodeToString(result)
	h := hmac.New(sha1.New, []byte(confAliyunOSS.AccessKeySecret))
	io.WriteString(h, encodedResult)
	signedStr := base64.StdEncoding.EncodeToString(h.Sum(nil))
	policyToken := &dto.PolicyTokenResponse{
		AccessKeyId: confAliyunOSS.AccessKeyId,
		Host:        host,
		Signature:   signedStr,
		Policy:      encodedResult,
		Directory:   uploadDir,
	}

	return policyToken, nil
}
