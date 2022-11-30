## Camunda-Input-Variables

### Module-Type
- Desc: sets Module.ModuleType; default is `config.CamundaWorkerTopic`
- Variable-Name-Template: `{{config.WorkerParamPrefix}}.module_type`
- Value-Type: string
- Example-Variable-Name: `info.module_type`
- Example-Variable-Value: `widget`

### Module-Data
- Desc: sets Module.ModuleData; default is `{}`
- Variable-Name-Template: `{{config.WorkerParamPrefix}}.module_data`
- Value-Type: `json.Marshal(map[string]interface{})`
- Example-Variable-Name: `info.module_data`
- Example-Variable-Value: `{"foo": 42}`
- Example-ModuleData: `{"foo": 42}`

### Additional Module-Data
- Desc: Optional; enabled/disabled by `config.enable_additional_module_data_fields`; sets fields for Module.ModuleData. The "config.WorkerParamPrefix" will be trimmed before used as Module.ModuleData field name. Values will be interpreted as JSON. If the value is not a valid JSON string, it will be used as plain string.
- Variable-Name-Template: `{{config.WorkerParamPrefix}}.{{fieldName}}`
- Value-Type: string (will be unmarshalled as JSON if possible)
- Example-Variable-Name: `info.button`
- Example-Variable-Value: `{"foo": 42}`
- Example-ModuleData: `{"button":{"foo": 42}}`