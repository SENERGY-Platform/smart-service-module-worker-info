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

package pkg

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/SENERGY-Platform/smart-service-module-worker-lib/pkg/configuration"
	"github.com/SENERGY-Platform/smart-service-module-worker-lib/pkg/model"
	"log"
	"strings"
)

type Config struct {
	WorkerParamPrefix                string `json:"worker_param_prefix"`
	EnableDeleteInfo                 bool   `json:"enable_delete_info"`
	EnableAdditionalModuleDataFields bool   `json:"enable_additional_module_data_fields"`
	Debug                            bool   `json:"debug"`
}

func New(config Config, libConfig configuration.Config) *Info {
	return &Info{config: config, libConfig: libConfig}
}

type Info struct {
	config    Config
	libConfig configuration.Config
}

func (this *Info) Do(task model.CamundaExternalTask) (modules []model.Module, outputs map[string]interface{}, err error) {
	info, err := this.getSmartServiceModuleInit(task)
	return []model.Module{{
			Id:                     task.ProcessInstanceId + "." + task.Id,
			ProcesInstanceId:       task.ProcessInstanceId,
			SmartServiceModuleInit: info,
		}},
		map[string]interface{}{},
		err
}

func (this *Info) Undo(modules []model.Module, reason error) {}

func (this *Info) getSmartServiceModuleInit(task model.CamundaExternalTask) (result model.SmartServiceModuleInit, err error) {
	if this.config.Debug {
		temp, _ := json.Marshal(task.Variables)
		log.Println("received task variables", string(temp))
	}
	moduleData, err := this.getModuleData(task)
	if this.config.EnableAdditionalModuleDataFields {
		for key, value := range this.getModuleDataAdditionalFields(task) {
			moduleData[key] = value
		}
	}
	return model.SmartServiceModuleInit{
		DeleteInfo: this.getDeleteInfo(task),
		ModuleType: this.getModuleType(task),
		ModuleData: moduleData,
	}, err
}

func (this *Info) getModuleType(task model.CamundaExternalTask) string {
	variable, ok := task.Variables[this.config.WorkerParamPrefix+"module_type"]
	if !ok {
		return this.libConfig.CamundaWorkerTopic
	}
	result, ok := variable.Value.(string)
	if !ok {
		return this.libConfig.CamundaWorkerTopic
	}
	return result
}

func (this *Info) getModuleData(task model.CamundaExternalTask) (result map[string]interface{}, err error) {
	variable, ok := task.Variables[this.config.WorkerParamPrefix+"module_data"]
	if !ok {
		if this.config.Debug {
			log.Println("no module_data found")
		}
		return map[string]interface{}{}, nil
	}
	temp, ok := variable.Value.(string)
	if !ok {
		if this.config.Debug {
			log.Println("module_data is not string")
		}
		return map[string]interface{}{}, errors.New("module_data is not string")
	}
	err = json.Unmarshal([]byte(temp), &result)
	if err != nil {
		log.Println("ERROR: module_data is not valid json", temp, err)
		return map[string]interface{}{}, fmt.Errorf(" module_data is not valid json: %w, (%v)", err, temp)
	}
	return result, nil
}

func (this *Info) getDeleteInfo(task model.CamundaExternalTask) (result *model.ModuleDeleteInfo) {
	if !this.config.EnableDeleteInfo {
		return nil
	}
	variable, ok := task.Variables[this.config.WorkerParamPrefix+"delete_info"]
	if !ok {
		return nil
	}
	temp, ok := variable.Value.(string)
	if !ok {
		return nil
	}
	err := json.Unmarshal([]byte(temp), result)
	if err != nil {
		return nil
	}
	result.UserId = ""
	return result
}

func (this *Info) getModuleDataAdditionalFields(task model.CamundaExternalTask) (result map[string]interface{}) {
	result = map[string]interface{}{}
	for key, value := range task.Variables {
		if strings.HasPrefix(key, this.config.WorkerParamPrefix) {
			key = strings.TrimPrefix(key, this.config.WorkerParamPrefix)
			str, ok := value.Value.(string)
			if !ok {
				break
			}
			if key != "module_data" && key != "module_type" && key != "delete_info" {
				var temp interface{}
				err := json.Unmarshal([]byte(str), &temp)
				if err != nil {
					result[key] = str
				} else {
					result[key] = temp
				}
			}
		}
	}
	return result
}
