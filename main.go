// Copyright 2021 FastWeGo
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/faabiosr/cachego/file"

	"github.com/fastwego/dingding"
	"github.com/spf13/viper"

	"github.com/gin-gonic/gin"
)

var DingClient *dingding.Client
var DingConfig map[string]string

func init() {
	// 加载配置文件
	viper.SetConfigFile(".env")
	_ = viper.ReadInConfig()

	DingConfig = map[string]string{
		"CorpId":         viper.GetString("CorpId"),
		"AgentId":        viper.GetString("AgentId"),
		"AppKey":         viper.GetString("AppKey"),
		"AppSecret":      viper.GetString("AppSecret"),
		"Token":          viper.GetString("TOKEN"),
		"EncodingAESKey": viper.GetString("EncodingAESKey"),
	}

	// 钉钉 AccessToken 管理器
	atm := &dingding.DefaultAccessTokenManager{
		Id:   DingConfig["AppKey"],
		Name: "access_token",
		GetRefreshRequestFunc: func() *http.Request {
			params := url.Values{}
			params.Add("appkey", DingConfig["AppKey"])
			params.Add("appsecret", DingConfig["AppSecret"])
			req, _ := http.NewRequest(http.MethodGet, dingding.ServerUrl+"/gettoken?"+params.Encode(), nil)

			return req
		},
		Cache: file.New(os.TempDir()),
	}

	// 钉钉 客户端
	DingClient = dingding.NewClient(atm)

	atm.Cache = file.New(os.TempDir())
}

func main() {

	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

	// 调用接口
	router.GET("/user/get_by_mobile", func(c *gin.Context) {
		params := url.Values{}
		params.Add("mobile", "13800138000")

		req, _ := http.NewRequest(http.MethodGet, "/user/get_by_mobile?"+params.Encode(), nil)
		get, err := DingClient.Do(req)

		if err != nil {
			log.Println(err)
			return
		}
		log.Println(string(get))
	})

	// 事件回调
	router.POST("/api/dingding/callback", Callback)

	// 文件上传
	router.GET("/api/upload/single", UploadSingle)
	router.GET("/api/upload/chunk", UploadChunk)

	svr := &http.Server{
		Addr:    viper.GetString("LISTEN"),
		Handler: router,
	}

	go func() {
		err := svr.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			log.Fatalln(err)
		}
	}()

	quit := make(chan os.Signal)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	timeout := time.Duration(5) * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := svr.Shutdown(ctx); err != nil {
		log.Fatalln(err)
	}
}
