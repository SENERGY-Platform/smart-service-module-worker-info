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

package tests

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	devicemodel "github.com/SENERGY-Platform/device-repository/v2/lib/model"
	"github.com/SENERGY-Platform/models/go/models"
	"github.com/SENERGY-Platform/smart-service-module-worker-info/pkg"
	"github.com/SENERGY-Platform/smart-service-module-worker-info/test/mocks"
	"github.com/SENERGY-Platform/smart-service-module-worker-lib/pkg/configuration"
	"github.com/SENERGY-Platform/smart-service-module-worker-lib/pkg/model"
)

const (
	testFunctionId = models.URN_PREFIX + "measuring-function:getTemperature"
	testAirAspect  = models.URN_PREFIX + "aspect:air"
	testRoomAspect = models.URN_PREFIX + "aspect:room"
)

// aspectIdConstants gives the scripts of these tests the ids as variables, so that the
// criteria under test stays readable next to the urns.
const aspectIdConstants = `
	var fid = "` + testFunctionId + `";
	var air = "` + testAirAspect + `";
	var room = "` + testRoomAspect + `";`

// A prescript of an info module names the aspects of a filter criteria in aspect_ids. The
// deprecated aspect_id stays usable as the alias for a list with one element: both spellings
// have to reach device-repository as the script wrote them, because device-repository folds
// the alias itself.
func TestPrescriptSendsAspectIdsToDeviceRepository(t *testing.T) {
	tests := []struct {
		name     string
		criteria string
		expected []devicemodel.FilterCriteria
	}{
		{
			name:     "an aspect list",
			criteria: `{function_id: fid, aspect_ids: [air, room]}`,
			expected: []devicemodel.FilterCriteria{{FunctionId: testFunctionId, AspectIds: []string{testAirAspect, testRoomAspect}}},
		},
		{
			name:     "an aspect list with one element",
			criteria: `{function_id: fid, aspect_ids: [air]}`,
			expected: []devicemodel.FilterCriteria{{FunctionId: testFunctionId, AspectIds: []string{testAirAspect}}},
		},
		{
			name:     "the deprecated single aspect id",
			criteria: `{function_id: fid, aspect_id: air}`,
			expected: []devicemodel.FilterCriteria{{FunctionId: testFunctionId, AspectId: testAirAspect}},
		},
		{
			name:     "both spellings side by side",
			criteria: `{function_id: fid, aspect_id: air, aspect_ids: [room]}`,
			expected: []devicemodel.FilterCriteria{{FunctionId: testFunctionId, AspectId: testAirAspect, AspectIds: []string{testRoomAspect}}},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			deviceRepo := mocks.NewDeviceRepoMock([]devicemodel.DeviceTypeSelectable{})
			runAspectTest(t, deviceRepo, aspectIdConstants+`
				deviceRepo.getDeviceTypeSelectables([`+test.criteria+`], "", true, true);`)

			expected := [][]devicemodel.FilterCriteria{test.expected}
			if !reflect.DeepEqual(deviceRepo.Criteria(), expected) {
				t.Errorf("device-repository was asked for %#v, not %#v", deviceRepo.Criteria(), expected)
			}
		})
	}
}

// The answering side: a path option names every matched aspect in aspect_nodes and keeps the
// alphabetically first of them in the deprecated aspect_node. A prescript reads both and hands
// them to the module data of the info module.
func TestPrescriptReadsAspectNodesIntoModuleData(t *testing.T) {
	deviceRepo := mocks.NewDeviceRepoMock([]devicemodel.DeviceTypeSelectable{{
		DeviceTypeId: "dt1",
		Services:     []models.Service{{Id: "service1"}},
		ServicePathOptions: map[string][]models.ServicePathOption{
			"service1": {{
				ServiceId:   "service1",
				Path:        "value.temperature",
				AspectNode:  models.AspectNode{Id: testAirAspect},
				AspectNodes: []models.AspectNode{{Id: testAirAspect}, {Id: testRoomAspect}},
			}},
		},
	}})

	smartServiceRepo := runAspectTest(t, deviceRepo, aspectIdConstants+`
		var selectables = deviceRepo.getDeviceTypeSelectables([{function_id: fid, aspect_ids: [air, room]}], "", true, true);
		var option = selectables[0].service_path_options["service1"][0];
		variables.write("aspects", JSON.stringify({
			"aspect_nodes": option.aspect_nodes.map(function(node) { return node.id; }),
			"aspect_node": option.aspect_node.id
		}));`)

	moduleData, err := getWrittenModuleData(smartServiceRepo)
	if err != nil {
		t.Error(err)
		return
	}
	expected := map[string]interface{}{
		"aspect_nodes": []interface{}{testAirAspect, testRoomAspect},
		"aspect_node":  testAirAspect,
	}
	if !reflect.DeepEqual(moduleData, expected) {
		t.Errorf("%#v", moduleData)
	}
}

// runAspectTest lets the info worker handle one task whose prescript is the given script and
// whose module data is the "aspects" variable that script may write.
func runAspectTest(t *testing.T, deviceRepo *mocks.DeviceRepoMock, script string) *mocks.SmartServiceRepoMock {
	t.Helper()
	libConf, err := configuration.LoadLibConfig("../config.json")
	if err != nil {
		t.Fatal(err)
	}
	conf, err := configuration.Load[pkg.Config]("../config.json")
	if err != nil {
		t.Fatal(err)
	}
	libConf.CamundaWorkerWaitDurationInMs = 200

	wg := &sync.WaitGroup{}
	t.Cleanup(wg.Wait)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	camunda := mocks.NewCamundaMock()
	libConf.CamundaUrl = camunda.Start(ctx, wg)
	camunda.AddToQueue([]model.CamundaExternalTask{{
		Id:                  "task1",
		ProcessInstanceId:   "process-instance-1",
		ProcessDefinitionId: "process-definition-1",
		Variables: map[string]model.CamundaVariable{
			"prescript":        {Value: script},
			"info.module_data": {Value: "{{.aspects}}"},
		},
	}})

	libConf.AuthEndpoint = mocks.Keycloak(ctx, wg)
	libConf.DeviceRepositoryUrl = deviceRepo.Start(ctx, wg)

	smartServiceRepo := mocks.NewSmartServiceRepoMock(libConf, conf, nil)
	libConf.SmartServiceRepositoryUrl = smartServiceRepo.Start(ctx, wg)

	err = pkg.Start(ctx, wg, conf, libConf)
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(1 * time.Second)
	return smartServiceRepo
}

func getWrittenModuleData(smartServiceRepo *mocks.SmartServiceRepoMock) (result map[string]interface{}, err error) {
	for _, request := range smartServiceRepo.GetRequestLog() {
		if request.Method == http.MethodPut && strings.Contains(request.Endpoint, "/modules/") {
			module := model.SmartServiceModuleInit{}
			err = json.Unmarshal([]byte(request.Message), &module)
			return module.ModuleData, err
		}
	}
	return nil, errors.New("no module was written")
}
