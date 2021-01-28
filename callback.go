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
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/fastwego/dingding"
	"github.com/gin-gonic/gin"
)

func Callback(c *gin.Context) {

	// 加解密处理器
	dingCrypto := dingding.NewCrypto(DingConfig["Token"], DingConfig["EncodingAESKey"], DingConfig["AppKey"])

	// Post Body
	bytes, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		return
	}

	log.Printf(string(bytes))

	msgJson := struct {
		Encrypt string `json:"encrypt"`
	}{}
	err = json.Unmarshal(bytes, &msgJson)
	if err != nil {
		return
	}

	timestamp := c.Request.URL.Query().Get("timestamp")
	nonce := c.Request.URL.Query().Get("nonce")
	signature := c.Request.URL.Query().Get("signature")
	decryptMsg, err := dingCrypto.GetDecryptMsg(timestamp, nonce, signature, msgJson.Encrypt)
	if err != nil {
		return
	}

	eventJson := struct {
		EventType string `json:"EventType"`
	}{}
	err = json.Unmarshal(decryptMsg, &eventJson)
	if err != nil {
		return
	}

	switch eventJson.EventType {
	default:
		// 响应 success
		encryptMsg := dingCrypto.GetEncryptMsg("success")
		c.JSON(http.StatusOK, encryptMsg)

		log.Println(encryptMsg)
	}
}
