/*
 * Copyright (c) 2026 InfAI (CC SES)
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
	"net/http"
	"net/http/httptest"
	"sync"

	devicemodel "github.com/SENERGY-Platform/device-repository/v2/lib/model"
	"github.com/julienschmidt/httprouter"
)

// DeviceRepoMock answers the device-type-selectables query over http. The in-memory client of
// device-repository stubs the selectables queries, so the criteria a script sends can only be
// observed at the http boundary.
type DeviceRepoMock struct {
	mux         sync.Mutex
	selectables []devicemodel.DeviceTypeSelectable
	criteria    [][]devicemodel.FilterCriteria
}

func NewDeviceRepoMock(selectables []devicemodel.DeviceTypeSelectable) *DeviceRepoMock {
	return &DeviceRepoMock{selectables: selectables}
}

// Criteria returns the decoded body of every device-type-selectables request, in order.
func (this *DeviceRepoMock) Criteria() [][]devicemodel.FilterCriteria {
	this.mux.Lock()
	defer this.mux.Unlock()
	return this.criteria
}

func (this *DeviceRepoMock) Start(ctx context.Context, wg *sync.WaitGroup) (url string) {
	server := httptest.NewServer(this.getRouter())
	wg.Add(1)
	go func() {
		<-ctx.Done()
		server.Close()
		wg.Done()
	}()
	return server.URL
}

func (this *DeviceRepoMock) getRouter() http.Handler {
	router := httprouter.New()
	router.POST("/v2/query/device-type-selectables", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		criteria := []devicemodel.FilterCriteria{}
		err := json.NewDecoder(request.Body).Decode(&criteria)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}
		this.mux.Lock()
		this.criteria = append(this.criteria, criteria)
		this.mux.Unlock()
		json.NewEncoder(writer).Encode(this.selectables)
	})
	return router
}
