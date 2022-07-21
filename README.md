## Camunda-Input-Variables

### Module-Data

- Desc: sets fields for Module.ModuleData. The "config.WorkerParamPrefix" will be trimmed before used as Module.ModuleData field name. Values will be interpreted as JSON. If the value is not a valid JSON string, it will be used as plain string.
- Variable-Name-Template: `{{config.WorkerParamPrefix}}.{{fieldName}}`
- Value-Type: string (will be unmarshalled as JSON if possible)
- Example-Variable-Name: `widget.button`
- Example-Variable-Value: `{"foo": 42}`
- Example-ModuleData: `{"button":{"foo": 42}}`
