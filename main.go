package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fastwego/dingding/apis/cspace"

	"github.com/fastwego/dingding/apis/microapp"

	"github.com/fastwego/dingding/types/event_types"

	"github.com/fastwego/dingding/apis/callback"

	"github.com/fastwego/dingding"
	"github.com/spf13/viper"

	"github.com/gin-gonic/gin"
)

var App *dingding.App

func init() {
	// 加载配置文件
	viper.SetConfigFile(".env")
	_ = viper.ReadInConfig()

	App = dingding.NewApp(dingding.AppConfig{
		CorpId:         viper.GetString("CorpId"),
		AgentId:        viper.GetString("AgentId"),
		AppKey:         viper.GetString("AppKey"),
		AppSecret:      viper.GetString("AppSecret"),
		Token:          viper.GetString("TOKEN"),
		EncodingAESKey: viper.GetString("EncodingAESKey"),
	})

}

func HandleEvent(c *gin.Context) {

	body, _ := ioutil.ReadAll(c.Request.Body)
	log.Println(string(body))

	event, err := App.Server.ParseEvent(body)
	if err != nil {
		log.Println(err)
	}

	var output interface{}

	switch event {
	case event_types.EventTypeCheckUrl:
		App.Server.CheckUrl(c.Writer, c.Request)
	}
	fmt.Println(output)

}

func HandleBot(c *gin.Context) {

	// 机器人 消息
	body, _ := ioutil.ReadAll(c.Request.Body)
	log.Println(string(body))

	msg := struct {
		Msgtype string `json:"msgtype"`
		Text    struct {
			Content string `json:"content"`
		} `json:"text"`
		MsgID    string `json:"msgId"`
		CreateAt int64  `json:"createAt"`
	}{}

	err := json.Unmarshal(body, &msg)
	if err != nil {
		return
	}

	// 回复 机器人
	reply := struct {
		Msgtype string `json:"msgtype"`
		Text    struct {
			Content string `json:"content"`
		} `json:"text"`
	}{}
	reply.Msgtype = "text"
	reply.Text.Content = msg.Text.Content

	fmt.Println(msg, reply)

	data, err := json.Marshal(reply)
	if err != nil {
		return
	}

	c.Writer.Write(data)
}

func main() {

	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

	router.GET("/api/demo", func(c *gin.Context) {

		resp, err := microapp.List(App, []byte(``))
		fmt.Println(string(resp), err)

		c.Writer.Write(resp)
	})

	router.GET("/api/upload", func(c *gin.Context) {

		params := url.Values{}
		params.Add("type", "image")
		resp, err := cspace.Upload(App, "qr2.png", params)
		fmt.Println(string(resp), err)

		c.Writer.Write(resp)
	})

	router.GET("/api/callback", func(c *gin.Context) {

		// 注册回调
		payload := struct {
			CallBackTag []string `json:"call_back_tag"`
			Token       string   `json:"token"`
			AesKey      string   `json:"aes_key"`
			URL         string   `json:"url"`
		}{
			CallBackTag: []string{"user_add_org", "user_modify_org", "user_leave_org"},
			Token:       App.Config.Token,
			AesKey:      App.Config.EncodingAESKey,
			URL:         viper.GetString("CallbackUrl"),
		}

		data, err := json.Marshal(payload)
		if err != nil {
			return
		}
		resp, err := callback.RegisterCallBack(App, data)
		fmt.Println(string(resp), err)
	})

	router.POST("/api/dingding", HandleEvent)
	router.POST("/api/dingding/bot", HandleBot)

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
