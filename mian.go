package main

/**
* s3上传图片
 */

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

var s3cfg S3cfg

type S3cfg struct {
	Endpoint  string `yaml:"endpoint"`
	Bucket    string `yaml:"bucket"`
	AccessKey string `yaml:"accesskey"`
	SecretKey string `yaml:"secretkey"`
	PathStyle bool   `yaml:"pathstyle"`
}

func init() {
	viper.AddConfigPath("./")
	viper.SetConfigName("app")
	viper.SetConfigType("yaml")
	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("read config file failed, %v", err)
	}
	if err := viper.Unmarshal(&s3cfg); err != nil {
		log.Fatalf("unmarshal config file failed, %v", err)
	}
}

func main() {
	r := gin.Default()

	r.POST("/upload", upload)

	r.Run(":8080")
}

func upload(ctx *gin.Context) {
	file, err := ctx.FormFile("file")
	if err != nil {
		ctx.JSON(http.StatusOK, gin.H{
			"code": 500,
			"msg":  "没有指定上传文件",
		})
		return
	}

	f, _ := file.Open()
	buf := make([]byte, file.Size)
	f.Read(buf)

	// bfRd := bufio.NewReader(f)
	// 上传对象
	err = uploadS3(s3cfg.Bucket, file.Filename, buf)
	if err != nil {
		ctx.JSON(http.StatusOK, gin.H{
			"code": "500",
			"msg":  "上传失败" + err.Error(),
		})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "文件上传成功",
	})
}

func newS3Client() *s3.S3 {
	creds := credentials.NewStaticCredentials(s3cfg.AccessKey, s3cfg.SecretKey, "")
	sess := session.Must(session.NewSession(&aws.Config{
		Region:           aws.String("ap-south-1"),
		Endpoint:         &s3cfg.Endpoint,
		S3ForcePathStyle: &s3cfg.PathStyle,
		Credentials:      creds,
	}))
	svc := s3.New(sess)
	return svc
}

func uploadS3(bucket, fileName string, body []byte) error {
	client := newS3Client()
	_, err := client.PutObjectWithContext(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(fileName),
		Body:   bytes.NewReader(body),
		ACL:    aws.String("public-read"),
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok && aerr.Code() == request.CanceledErrorCode {
			// If the SDK can determine the request or retry delay was canceled
			// by a context the CanceledErrorCode error code will be returned.
			return fmt.Errorf("upload canceled due to timeout, %v", err)
		}
		return fmt.Errorf("failed to upload object, %v", err)
	}
	return nil
}