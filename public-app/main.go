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
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/fastwego/dingding/util"

	"github.com/fastwego/dingding"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

var DingClient *dingding.Client
var DingClientSuite *dingding.Client

var DingConfig map[string]string

func init() {
	// 加载配置文件
	viper.SetConfigFile(".env")
	_ = viper.ReadInConfig()

	DingConfig = map[string]string{
		"SuiteKey":    viper.GetString("SuiteKey"),
		"SuiteSecret": viper.GetString("SuiteSecret"),
	}

	// 自定义获取 auth_corpid
	authCorpId := func() (corpId string) {
		return "authCorpId"
	}()

	// 钉钉 AccessToken 管理器
	atm := dingding.NewAccessTokenManager(DingConfig["SuiteKey"]+":"+authCorpId, "access_token", func() *http.Request {
		// 自定义获取 suiteTicket

		suiteTicket := func() (suiteTicket string) {
			return "suiteTicket"
		}()
		params := map[string]string{
			"accessKey":    DingConfig["SuiteKey"],
			"accessSecret": DingConfig["SuiteSecret"],
			"suiteTicket":  suiteTicket,
			"signature":    util.Signature(suiteTicket, DingConfig["SuiteSecret"]),
			"auth_corpid":  authCorpId,
		}
		data, err := json.Marshal(params)
		if err != nil {
			panic(err)
		}

		log.Printf(string(data))

		req, _ := http.NewRequest(http.MethodPost, dingding.ServerUrl+"/service/get_corp_token", bytes.NewReader(data))

		return req
	})
	// 钉钉 客户端
	DingClient = dingding.NewClient(atm)

	// 钉钉 SuiteAccessToken 管理器
	satm := dingding.NewAccessTokenManager(DingConfig["SuiteKey"], "suite_access_token", func() *http.Request {
		params := map[string]string{
			"suite_key":    DingConfig["SuiteKey"],
			"suite_secret": DingConfig["SuiteSecret"],
			"suite_ticket": func() (suiteTicket string) {
				return "suiteTicket"
			}(),
		}
		data, err := json.Marshal(params)
		if err != nil {
			panic(err)
		}

		log.Printf(string(data))

		req, _ := http.NewRequest(http.MethodPost, dingding.ServerUrl+"/service/get_suite_token", bytes.NewReader(data))

		return req
	})

	DingClientSuite = dingding.NewClient(satm)

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

	// 激活应用
	router.GET("/service/activate_suite", func(c *gin.Context) {

		payload := `{
        "auth_corpid":"ding1234",
        "suite_key":"suitexxxx"
}`
		req, _ := http.NewRequest(http.MethodPost, "/service/activate_suite", strings.NewReader(payload))
		get, err := DingClientSuite.Do(req)
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
