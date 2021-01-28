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

func UploadSingle(c *gin.Context) {

	// single upload
	uploadFile := "qr2.png"
	readme, err := os.Stat(uploadFile)
	if err != nil {
		fmt.Println(err)
		return
	}

	fileSize := strconv.FormatInt(readme.Size(), 10)

	params := url.Values{}
	params.Add("agent_id", DingConfig["AgentId"])
	params.Add("file_size", fileSize)

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

	req, _ := http.NewRequest(http.MethodPost, "/file/upload/single?"+params.Encode(), r)
	req.Header.Set("Content-Type", m.FormDataContentType())
	resp, err := DingClient.Do(req)

	if err != nil {
		log.Println(err)
		return
	}
	log.Println(string(resp))
}
