/*
 * Copyright (c) 2022 InfAI (CC SES)
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package mocks

import (
	"context"
	"encoding/json"
	"github.com/SENERGY-Platform/smart-service-module-worker-lib/pkg/model"
	"github.com/julienschmidt/httprouter"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
)

type Request struct {
	Method   string `json:"method"`
	Endpoint string `json:"endpoint"`
	Message  string `json:"message"`
}

type Response struct {
	StatusCode int    `json:"status_code"`
	Message    string `json:"message"`
}

func NewCamundaMock() *CamundaMock {
	return &CamundaMock{
		Queue: make(chan []model.CamundaExternalTask, 20),
	}
}

type CamundaMock struct {
	Queue       chan []model.CamundaExternalTask
	requestsLog []Request
	mux         sync.Mutex
}

func (this *CamundaMock) Fetch() (result []model.CamundaExternalTask) {
	select {
	case result = <-this.Queue:
		return result
	default:
		return []model.CamundaExternalTask{}
	}
}

func (this *CamundaMock) AddToQueue(fetchResult []model.CamundaExternalTask) {
	this.Queue <- fetchResult
}

func (this *CamundaMock) AddFileToQueue(location string) error {
	tasksFile, err := os.ReadFile(location)
	if err != nil {
		return err
	}
	var tasks []model.CamundaExternalTask
	err = json.Unmarshal(tasksFile, &tasks)
	if err != nil {
		return err
	}
	this.AddToQueue(tasks)
	return nil
}

func (this *CamundaMock) PopRequestLog() []Request {
	this.mux.Lock()
	defer this.mux.Unlock()
	result := this.requestsLog
	this.requestsLog = []Request{}
	return result
}

func (this *CamundaMock) logRequest(r Request) {
	this.mux.Lock()
	defer this.mux.Unlock()
	this.requestsLog = append(this.requestsLog, r)
}

func (this *CamundaMock) Start(ctx context.Context, wg *sync.WaitGroup) (url string) {
	server := httptest.NewServer(this.getRouter())
	wg.Add(1)
	go func() {
		<-ctx.Done()
		server.Close()
		wg.Done()
	}()
	return server.URL
}

func (this *CamundaMock) getRouter() http.Handler {
	router := httprouter.New()

	router.POST("/engine-rest/external-task/:taskId", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		if params.ByName("taskId") == "fetchAndLock" {
			list := this.Fetch()
			json.NewEncoder(writer).Encode(list)
			return
		} else {
			temp, _ := io.ReadAll(request.Body)
			this.logRequest(Request{
				Method:   request.Method,
				Endpoint: request.URL.Path,
				Message:  string(temp),
			})
			writer.WriteHeader(200)
		}
	})

	router.POST("/engine-rest/external-task/:taskId/complete", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		temp, _ := io.ReadAll(request.Body)
		this.logRequest(Request{
			Method:   request.Method,
			Endpoint: request.URL.Path,
			Message:  string(temp),
		})
		writer.WriteHeader(200)
	})

	router.DELETE("/engine-rest/process-instance/:id", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		temp, _ := io.ReadAll(request.Body)
		this.logRequest(Request{
			Method:   request.Method,
			Endpoint: request.URL.Path,
			Message:  string(temp),
		})
		writer.WriteHeader(200)
	})

	return router
}
