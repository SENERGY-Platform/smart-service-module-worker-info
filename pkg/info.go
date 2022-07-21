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
	"github.com/SENERGY-Platform/smart-service-module-worker-lib/pkg/configuration"
	"github.com/SENERGY-Platform/smart-service-module-worker-lib/pkg/model"
	"strings"
)

type Config struct {
	WorkerParamPrefix string `json:"worker_param_prefix"`
}

func New(config Config, libConfig configuration.Config) *Info {
	return &Info{config: config, libConfig: libConfig}
}

type Info struct {
	config    Config
	libConfig configuration.Config
}

func (this *Info) Do(task model.CamundaExternalTask) (modules []model.Module, outputs map[string]interface{}, err error) {
	return []model.Module{{
			Id:               task.ProcessInstanceId + "." + task.Id,
			ProcesInstanceId: task.ProcessInstanceId,
			SmartServiceModuleInit: model.SmartServiceModuleInit{
				ModuleType: this.libConfig.CamundaWorkerTopic,
				ModuleData: this.getModuleData(task),
			},
		}},
		map[string]interface{}{},
		err
}

func (this *Info) Undo(modules []model.Module, reason error) {}

func (this *Info) getModuleData(task model.CamundaExternalTask) (result map[string]interface{}) {
	result = map[string]interface{}{}
	for key, value := range task.Variables {
		if strings.HasPrefix(key, this.config.WorkerParamPrefix) {
			key = strings.TrimPrefix(key, this.config.WorkerParamPrefix)
			str, ok := value.Value.(string)
			if !ok {
				break
			}
			var temp interface{}
			err := json.Unmarshal([]byte(str), &temp)
			if err != nil {
				result[key] = str
			} else {
				result[key] = temp
			}
		}

	}
	return result
}
