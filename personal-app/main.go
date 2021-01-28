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

	"github.com/fastwego/dingding"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

var DingClient *dingding.Client
var DingConfig map[string]string

func init() {
	// 加载配置文件
	viper.SetConfigFile(".env")
	_ = viper.ReadInConfig()

	DingConfig = map[string]string{
		"AppId":     viper.GetString("AppId"),
		"AppSecret": viper.GetString("AppSecret"),
	}

	// 钉钉 AccessToken 管理器
	atm := dingding.NewAccessTokenManager(viper.GetString("AppKey"), "access_token", func() *http.Request {
		params := url.Values{}
		params.Add("appid", DingConfig["AppId"])
		params.Add("appsecret", DingConfig["AppSecret"])

		req, _ := http.NewRequest(http.MethodGet, dingding.ServerUrl+"/sns/gettoken?"+params.Encode(), nil)
		return req
	})

	// 钉钉 客户端
	DingClient = dingding.NewClient(atm)

}
func main() {

	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

	// 调用接口
	router.GET("/sns/get_persistent_code", func(c *gin.Context) {
		params := url.Values{}
		params.Add("tmp_auth_code", "13800138000")

		req, _ := http.NewRequest(http.MethodGet, "/sns/get_persistent_code?"+params.Encode(), nil)
		get, err := DingClient.Do(req)
		if err != nil {
			log.Println(err)
			return
		}
		log.Println(string(get))
	})

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
