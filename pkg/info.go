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
	"sort"
	"strings"
)

type Config struct {
	WorkerParamPrefix                string `json:"worker_param_prefix"`
	EnableAdditionalModuleDataFields bool   `json:"enable_additional_module_data_fields"`
	Debug                            bool   `json:"debug"`
}

func New(config Config, libConfig configuration.Config, repo SmartServiceRepo) *Info {
	return &Info{config: config, libConfig: libConfig, smartServiceRepo: repo}
}

type Info struct {
	config           Config
	libConfig        configuration.Config
	smartServiceRepo SmartServiceRepo
}

type SmartServiceRepo interface {
	GetInstanceUser(instanceId string) (userId string, err error)
	UseModuleDeleteInfo(info model.ModuleDeleteInfo) error
	ListExistingModules(processInstanceId string, query model.ModulQuery) (result []model.SmartServiceModule, err error)
}

func (this *Info) Do(task model.CamundaExternalTask) (modules []model.Module, outputs map[string]interface{}, err error) {
	key := this.getModuleKey(task)
	if key == nil {
		return this.createModule(task, []string{})
	} else {
		existingModule, exists, err := this.getExistingModule(task.ProcessInstanceId, *key)
		if err != nil {
			return nil, nil, err
		}
		if !exists {
			return this.createModule(task, []string{*key})
		} else {
			return this.updateModule(task, existingModule, []string{*key})
		}
	}
}

func (this *Info) createModule(task model.CamundaExternalTask, keys []string) ([]model.Module, map[string]interface{}, error) {
	info, err := this.getSmartServiceModuleInit(task)
	info.Keys = keys
	return []model.Module{{
			Id:                     task.ProcessInstanceId + "." + task.Id,
			ProcesInstanceId:       task.ProcessInstanceId,
			SmartServiceModuleInit: info,
		}},
		map[string]interface{}{},
		err
}

func (this *Info) updateModule(task model.CamundaExternalTask, existingModule model.Module, keys []string) ([]model.Module, map[string]interface{}, error) {
	info, err := this.getSmartServiceModuleInit(task)
	if err != nil {
		return nil, nil, err
	}
	info.Keys = keys
	existingModule.SmartServiceModuleInit = info
	return []model.Module{existingModule},
		map[string]interface{}{},
		nil
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

type KeyValue struct {
	Key   string
	Value string
}

func (this *Info) getModuleData(task model.CamundaExternalTask) (result map[string]interface{}, err error) {
	parts := []KeyValue{}
	for key, variable := range task.Variables {
		if strings.HasPrefix(key, this.config.WorkerParamPrefix+"module_data") {
			temp, ok := variable.Value.(string)
			if !ok {
				if this.config.Debug {
					log.Println("module_data is not string")
				}
				return map[string]interface{}{}, errors.New("module_data is not string")
			}
			parts = append(parts, KeyValue{
				Key:   key,
				Value: temp,
			})
		}
	}
	if len(parts) == 0 {
		if this.config.Debug {
			log.Println("no module_data found")
		}
		return map[string]interface{}{}, nil
	}
	sort.Slice(parts, func(i, j int) bool {
		return parts[i].Key < parts[j].Key
	})
	joined := ""
	for _, part := range parts {
		joined = joined + part.Value
	}
	err = json.Unmarshal([]byte(joined), &result)
	if err != nil {
		log.Println("ERROR: module_data is not valid json", joined, err)
		return map[string]interface{}{}, fmt.Errorf(" module_data is not valid json: %w, (%v)", err, joined)
	}
	return result, nil
}

func (this *Info) getModuleDataAdditionalFields(task model.CamundaExternalTask) (result map[string]interface{}) {
	result = map[string]interface{}{}
	for key, value := range task.Variables {
		if strings.HasPrefix(key, this.config.WorkerParamPrefix) && !strings.HasPrefix(key, this.config.WorkerParamPrefix+"module_data") {
			key = strings.TrimPrefix(key, this.config.WorkerParamPrefix)
			str, ok := value.Value.(string)
			if !ok {
				break
			}
			if key != "module_data" && key != "module_type" && key != "delete_info" && key != "key" {
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

// if no key is set: return nil
func (this *Info) getModuleKey(task model.CamundaExternalTask) (key *string) {
	variable, ok := task.Variables[this.config.WorkerParamPrefix+"key"]
	if !ok {
		return nil
	}
	result, ok := variable.Value.(string)
	if ok && result != "" {
		return &result
	}
	return nil
}

func (this *Info) getExistingModule(processInstanceId string, key string) (module model.Module, exists bool, err error) {
	existingModules, err := this.smartServiceRepo.ListExistingModules(processInstanceId, model.ModulQuery{
		KeyFilter: &key,
	})
	if err != nil {
		log.Println("ERROR:", err)
		return module, false, err
	}
	if this.config.Debug {
		log.Printf("DEBUG: existing module request: %v, %v, \n %#v", processInstanceId, key, existingModules)
	}
	if len(existingModules) == 0 {
		return module, false, nil
	}
	if len(existingModules) > 1 {
		log.Printf("WARNING: more than one existing module found: %v, %v, \n %#v", processInstanceId, key, existingModules)
	}
	module.SmartServiceModuleInit = existingModules[0].SmartServiceModuleInit
	module.ProcesInstanceId = processInstanceId
	module.Id = existingModules[0].Id
	return module, true, nil
}
