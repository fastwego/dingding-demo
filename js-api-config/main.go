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
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/faabiosr/cachego/file"

	"github.com/fastwego/dingding"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"

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

	// Session
	store := cookie.NewStore([]byte("secret"))
	router.Use(sessions.Sessions("gosession", store))

	router.GET("/", Index)

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

type User struct {
	JsApiConfig template.JS `json:"js_api_config"`
}

func Index(c *gin.Context) {

	loginUser := User{}

	var err error
	config, err := jsapiConfig(c)
	if err != nil {
		fmt.Println(err)
		return
	}
	loginUser.JsApiConfig = template.JS(config)

	t1, err := template.ParseFiles("index.html")
	if err != nil {
		fmt.Println(err)
		return
	}

	t1.Execute(c.Writer, loginUser)
}

func jsapiConfig(c *gin.Context) (config string, err error) {

	TicketResp := struct {
		Errcode   int    `json:"errcode"`
		Errmsg    string `json:"errmsg"`
		Ticket    string `json:"ticket"`
		ExpiresIn int    `json:"expires_in"`
	}{}

	req, _ := http.NewRequest(http.MethodGet, "/get_jsapi_ticket", nil)
	getJSApiTicket, err := DingClient.Do(req)
	if err != nil {
		return
	}
	err = json.Unmarshal(getJSApiTicket, &TicketResp)
	if err != nil {
		return
	}
	fmt.Println(TicketResp)

	nonceStr := "hello"
	timeStamp := strconv.FormatInt(time.Now().Unix(), 10)
	pageUrl := "http://" + c.Request.Host + c.Request.RequestURI
	plain := "jsapi_ticket=" + TicketResp.Ticket + "&noncestr=" + nonceStr + "&timestamp=" + timeStamp + "&url=" + pageUrl
	signature := fmt.Sprintf("%x", sha1.Sum([]byte(plain)))

	configMap := map[string]string{
		"url":       pageUrl,
		"nonceStr":  nonceStr,
		"agentId":   DingConfig["AgentId"],
		"timeStamp": timeStamp,
		"corpId":    DingConfig["CorpId"],
		"signature": signature,
	}

	marshal, err := json.Marshal(configMap)
	if err != nil {
		return
	}

	return string(marshal), nil
}
