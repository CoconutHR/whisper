package blob

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type R2Config struct {
	AccountID       string
	AccessKeyID     string
	SecretAccessKey string
	Bucket          string
}

type R2Store struct {
	bucket    string
	origin    string
	client    *s3.Client
	presigner *s3.PresignClient
}

func R2ConfigFromEnv() (R2Config, bool, error) {
	config := R2Config{
		AccountID:       strings.TrimSpace(os.Getenv("WHISPER_R2_ACCOUNT_ID")),
		AccessKeyID:     strings.TrimSpace(os.Getenv("WHISPER_R2_ACCESS_KEY_ID")),
		SecretAccessKey: strings.TrimSpace(os.Getenv("WHISPER_R2_SECRET_ACCESS_KEY")),
		Bucket:          strings.TrimSpace(os.Getenv("WHISPER_R2_BUCKET")),
	}
	values := []string{config.AccountID, config.AccessKeyID, config.SecretAccessKey, config.Bucket}
	configured := false
	complete := true
	for _, value := range values {
		configured = configured || value != ""
		complete = complete && value != ""
	}
	if !configured {
		return R2Config{}, false, nil
	}
	if !complete {
		return R2Config{}, false, errors.New("R2 配置不完整，需要同时设置 WHISPER_R2_ACCOUNT_ID、WHISPER_R2_ACCESS_KEY_ID、WHISPER_R2_SECRET_ACCESS_KEY 和 WHISPER_R2_BUCKET")
	}
	return config, true, nil
}

func NewR2Store(ctx context.Context, config R2Config) (*R2Store, error) {
	origin := fmt.Sprintf("https://%s.r2.cloudflarestorage.com", config.AccountID)
	awsConfig, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion("auto"),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			config.AccessKeyID, config.SecretAccessKey, "",
		)),
	)
	if err != nil {
		return nil, fmt.Errorf("初始化 R2 SDK: %w", err)
	}
	client := s3.NewFromConfig(awsConfig, func(options *s3.Options) {
		options.BaseEndpoint = aws.String(origin)
		options.UsePathStyle = true
	})
	return &R2Store{
		bucket: config.Bucket, origin: origin, client: client, presigner: s3.NewPresignClient(client),
	}, nil
}

func (s *R2Store) UploadOrigin() string {
	return s.origin
}

func (s *R2Store) PresignPut(ctx context.Context, key, contentType string, size int64, ttl time.Duration) (PresignedRequest, error) {
	result, err := s.presigner.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.bucket), Key: aws.String(key), ContentType: aws.String(contentType),
		ContentLength: aws.Int64(size),
	}, s3.WithPresignExpires(ttl))
	if err != nil {
		return PresignedRequest{}, fmt.Errorf("生成 R2 上传地址: %w", err)
	}
	return PresignedRequest{
		URL: result.URL, Headers: map[string]string{"Content-Type": contentType}, ExpiresAt: time.Now().Add(ttl),
	}, nil
}

func (s *R2Store) PresignGet(ctx context.Context, key, contentDisposition string, ttl time.Duration) (PresignedRequest, error) {
	result, err := s.presigner.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket), Key: aws.String(key),
		ResponseContentDisposition: aws.String(contentDisposition),
	}, s3.WithPresignExpires(ttl))
	if err != nil {
		return PresignedRequest{}, fmt.Errorf("生成 R2 下载地址: %w", err)
	}
	return PresignedRequest{URL: result.URL, ExpiresAt: time.Now().Add(ttl)}, nil
}

func (s *R2Store) Head(ctx context.Context, key string) (ObjectMetadata, error) {
	result, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket), Key: aws.String(key),
	})
	if err != nil {
		return ObjectMetadata{}, fmt.Errorf("读取 R2 对象信息: %w", err)
	}
	return ObjectMetadata{Size: aws.ToInt64(result.ContentLength), ContentType: aws.ToString(result.ContentType)}, nil
}

func (s *R2Store) Delete(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket), Key: aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("删除 R2 对象: %w", err)
	}
	return nil
}
