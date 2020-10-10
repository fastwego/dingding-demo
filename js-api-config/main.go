package main

import (
	"context"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/fastwego/dingding"
	"github.com/fastwego/dingding/apis/auth"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
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
	Userid string `json:"userid"`
	Name   string `json:"name"`
	Avatar string `json:"avatar"`

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

	session := sessions.Default(c)
	jsapiConfig := session.Get("jsapiConfig")
	jsapiConfig2, ok := jsapiConfig.(string)
	if ok && len(jsapiConfig2) > 0 {
		fmt.Println("cache jsapiConfig ", jsapiConfig2)
		return jsapiConfig2, nil
	}

	TicketResp := struct {
		Errcode   int    `json:"errcode"`
		Errmsg    string `json:"errmsg"`
		Ticket    string `json:"ticket"`
		ExpiresIn int    `json:"expires_in"`
	}{}

	getJSApiTicket, err := auth.GetJSApiTicket(App)
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
		"agentId":   App.Config.AgentId,
		"timeStamp": timeStamp,
		"corpId":    App.Config.CorpId,
		"signature": signature,
	}

	marshal, err := json.Marshal(configMap)
	if err != nil {
		return
	}

	session.Set("jsapiConfig", string(marshal))
	err = session.Save()
	if err != nil {
		return
	}

	return string(marshal), nil
}
