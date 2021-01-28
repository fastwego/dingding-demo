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
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"

	"github.com/gin-gonic/gin"
)

func UploadChunk(c *gin.Context) {

	// 分块最小需大于100KB，最大不超过8M，最多支持10000块。
	uploadFile := "tmp.200k"
	readme, err := os.Stat(uploadFile)
	if err != nil {
		fmt.Println(err)
		return
	}

	if readme.Size() < 200*1024 {
		fmt.Println("readme.Size() < 200 * 1024")
		return
	}

	fileSize := strconv.FormatInt(2*readme.Size(), 10)

	params := url.Values{}
	params.Add("agent_id", DingConfig["AgentId"])
	params.Add("file_size", fileSize)
	params.Add("chunk_numbers", "2")

	req, _ := http.NewRequest(http.MethodGet, "/file/upload/transaction?"+params.Encode(), nil)
	data, err := DingClient.Do(req)
	if err != nil {
		log.Println(err)
		return
	}

	tx := struct {
		UploadID string `json:"upload_id"`
	}{}

	err = json.Unmarshal(data, &tx)
	if err != nil {
		log.Println(err)
		return
	}

	// 文件 1
	chunk("1", tx.UploadID, uploadFile)

	// 文件 2
	chunk("2", tx.UploadID, uploadFile)

	// 提交事务
	params = url.Values{}
	params.Add("agent_id", DingConfig["AgentId"])
	params.Add("file_size", fileSize)
	params.Add("chunk_numbers", "2")
	params.Add("upload_id", tx.UploadID)

	req, _ = http.NewRequest(http.MethodGet, "/file/upload/transaction?"+params.Encode(), nil)
	data, err = DingClient.Do(req)

	log.Println(string(data), err)
	if err != nil {
		return
	}

	c.Writer.Write(data)
}

func chunk(seq string, uploadId string, uploadFile string) {
	params := url.Values{}
	params.Add("agent_id", DingConfig["AgentId"])
	params.Add("chunk_sequence", seq)
	params.Add("upload_id", uploadId)

	r, w := io.Pipe()
	m := multipart.NewWriter(w)
	go func() {
		defer w.Close()
		defer m.Close()
		part, err := m.CreateFormFile("media", path.Base(uploadFile))
		if err != nil {
			return
		}
		file, err := os.Open(uploadFile)
		if err != nil {
			return
		}
		defer file.Close()
		if _, err = io.Copy(part, file); err != nil {
			return
		}

	}()

	req, _ := http.NewRequest(http.MethodPost, "/file/upload/chunk?"+params.Encode(), r)
	req.Header.Set("Content-Type", m.FormDataContentType())
	data, err := DingClient.Do(req)

	fmt.Println(string(data), err)
	if err != nil {
		return
	}
}
