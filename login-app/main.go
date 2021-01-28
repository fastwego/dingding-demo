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
	"encoding/gob"
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
	router.POST("/login", Login)

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
	Userid string `json:"userid"`
	Name   string `json:"name"`
	Avatar string `json:"avatar"`

	Message string `json:"message"`
	CorpId  string `json:"corp_id"`
}

func Index(c *gin.Context) {

	session := sessions.Default(c)
	user := session.Get("user")

	loginUser, ok := user.(User)
	if !ok {
		loginUser = User{CorpId: DingConfig["CorpId"]}
	}

	join := c.Query("join")
	if len(join) > 0 {
		// 发送 报名信息
		type Msg struct {
			Msgtype    string `json:"msgtype"`
			ActionCard struct {
				Title       string `json:"title"`
				Markdown    string `json:"markdown"`
				SingleTitle string `json:"single_title"`
				SingleURL   string `json:"single_url"`
			} `json:"action_card"`
		}
		msg := Msg{}
		msg.Msgtype = "action_card"
		msg.ActionCard.Title = "报名成功@" + strconv.FormatInt(time.Now().Unix(), 10)
		msg.ActionCard.Markdown = `![alt](https://pic3.zhimg.com/50/v2-b7927e012c63682d0a93fba30b3ee419_hd.jpg?source=1940ef5c) 
## 今天晚上不见不散
`
		msg.ActionCard.SingleTitle = "马上看看"
		msg.ActionCard.SingleURL = "https://fastwego.dev"

		data := struct {
			AgentId    string `json:"agent_id"`
			UseridList string `json:"userid_list"`
			Msg        Msg    `json:"msg"`
		}{
			AgentId:    DingConfig["AgentId"],
			UseridList: loginUser.Userid,
		}
		data.Msg = msg

		payload, err := json.Marshal(data)
		fmt.Println(string(payload), err)
		if err != nil {
			return
		}

		req, _ := http.NewRequest(http.MethodPost, "/topapi/message/corpconversation/asyncsend_v2", bytes.NewReader(payload))
		resp, err := DingClient.Do(req)

		fmt.Println(string(resp), err)

		loginUser.Message = "报名成功~"
	}

	t1, err := template.ParseFiles("index.html")
	if err != nil {
		fmt.Println(err)
		return
	}

	t1.Execute(c.Writer, loginUser)
}

func Login(c *gin.Context) {

	code := c.PostForm("code")
	fmt.Println("code = ", code)

	if len(code) == 0 {
		return
	}

	// 获取用户身份
	params := url.Values{}
	params.Add("code", code)

	req, _ := http.NewRequest(http.MethodGet, "/user/getuserinfo?"+params.Encode(), nil)
	userInfo, err := DingClient.Do(req)
	log.Println(string(userInfo), err)
	if err != nil {
		return
	}

	UserInfo := struct {
		Userid   string `json:"userid"`
		SysLevel int    `json:"sys_level"`
		Errmsg   string `json:"errmsg"`
		IsSys    bool   `json:"is_sys"`
		Errcode  int    `json:"errcode"`
	}{}

	err = json.Unmarshal(userInfo, &UserInfo)
	if err != nil {
		fmt.Println(err)
		return
	}

	// 获取员工详细信息
	params = url.Values{}
	params.Add("userid", UserInfo.Userid)
	req, _ = http.NewRequest(http.MethodGet, "/user/get?"+params.Encode(), nil)
	resp, err := DingClient.Do(req)
	fmt.Println(string(resp), err)
	if err != nil {
		return
	}

	user := User{}

	err = json.Unmarshal(resp, &user)
	if err != nil {
		fmt.Println(err)
		return
	}
	user.CorpId = DingConfig["CorpId"]

	// 记录 Session
	gob.Register(User{})
	session := sessions.Default(c)
	session.Set("user", user)
	fmt.Println(user)
	err = session.Save()

	if err != nil {
		fmt.Println(err)
		return
	}

	c.JSON(200, user)
}
