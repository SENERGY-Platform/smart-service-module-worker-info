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
	"fmt"
	"github.com/SENERGY-Platform/smart-service-module-worker-info/pkg"
	"github.com/SENERGY-Platform/smart-service-module-worker-lib/pkg/configuration"
	"github.com/julienschmidt/httprouter"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"sync"
)

func NewSmartServiceRepoMock(libConfig configuration.Config, config pkg.Config, moduleListResponse []byte) *SmartServiceRepoMock {
	return &SmartServiceRepoMock{libConfig: libConfig, config: config, moduleListResponse: moduleListResponse}
}

type SmartServiceRepoMock struct {
	requestsLog        []Request
	mux                sync.Mutex
	libConfig          configuration.Config
	config             pkg.Config
	moduleListResponse []byte
}

func (this *SmartServiceRepoMock) PopRequestLog() []Request {
	this.mux.Lock()
	defer this.mux.Unlock()
	result := this.requestsLog
	this.requestsLog = []Request{}
	return result
}

func (this *SmartServiceRepoMock) GetRequestLog() []Request {
	this.mux.Lock()
	defer this.mux.Unlock()
	result := this.requestsLog
	return result
}

func (this *SmartServiceRepoMock) logRequest(r Request) {
	this.mux.Lock()
	defer this.mux.Unlock()
	this.requestsLog = append(this.requestsLog, r)
}

func (this *SmartServiceRepoMock) Start(ctx context.Context, wg *sync.WaitGroup) (url string) {
	server := httptest.NewServer(this.getRouter())
	wg.Add(1)
	go func() {
		<-ctx.Done()
		server.Close()
		wg.Done()
	}()
	return server.URL
}

func (this *SmartServiceRepoMock) getRouter() http.Handler {
	router := httprouter.New()

	router.PUT("/instances-by-process-id/:id/error", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		temp, _ := io.ReadAll(request.Body)
		this.logRequest(Request{
			Method:   request.Method,
			Endpoint: request.URL.Path,
			Message:  string(temp),
		})
		writer.WriteHeader(200)
	})

	router.PUT("/instances-by-process-id/:id/modules/:moduleId", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		temp, _ := io.ReadAll(request.Body)
		msg := string(temp)
		this.logRequest(Request{
			Method:   request.Method,
			Endpoint: request.URL.Path,
			Message:  msg,
		})
		writer.Write(temp)
	})

	router.GET("/instances-by-process-id/:id/modules", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		temp, _ := io.ReadAll(request.Body)
		msg := string(temp)
		this.logRequest(Request{
			Method:   request.Method,
			Endpoint: request.URL.Path + "?" + request.URL.Query().Encode(),
			Message:  msg,
		})
		writer.Write(this.moduleListResponse)
	})

	router.GET("/instances-by-process-id/:id/user-id", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		temp, _ := io.ReadAll(request.Body)
		this.logRequest(Request{
			Method:   request.Method,
			Endpoint: request.URL.Path,
			Message:  string(temp),
		})
		json.NewEncoder(writer).Encode(userId)
	})

	return router
}

func (this *SmartServiceRepoMock) CheckExpectedRequests(expectedRequests []Request) error {
	actualEngineRequests := this.GetRequestLog()
	if !reflect.DeepEqual(expectedRequests, actualEngineRequests) {
		a, _ := json.Marshal(actualEngineRequests)
		e, _ := json.Marshal(expectedRequests)
		return fmt.Errorf("\n %v \n %v", string(a), string(e))
	}
	return nil
}

func (this *SmartServiceRepoMock) CheckExpectedRequestsFromFileLocation(fileLocation string) error {
	fileContent, err := os.ReadFile(fileLocation)
	if err != nil {
		return err
	}
	expectedRequests := []Request{}
	err = json.Unmarshal(fileContent, &expectedRequests)
	if err != nil {
		return err
	}
	return this.CheckExpectedRequests(expectedRequests)
}
