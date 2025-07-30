package fptcloud_instance

import "github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"


func GetStringOrEmpty(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok && v != nil {
		return v.(string)
	}
	return ""
}

func GetValueOrEmpty(m map[string]interface{}, key string) interface{} {
	if v, ok := m[key]; ok && v != nil {
		return v
	}
	return nil
}

func ParseActionBlock(v interface{}) *ActionBlock {
	if v == nil {
		return nil
	}
	m := v.(map[string]interface{})
	return &ActionBlock{
		Type:        GetStringOrEmpty(m, "type"),
		Name:        GetStringOrEmpty(m, "name"),
		Description: GetStringOrEmpty(m, "description"),
		IncludeRam:  GetValueOrEmpty(m, "include_ram"),
	}
}

// ParseVMAction parses VM action from schema data
func ParseVMAction(v interface{}) *VMAction {
	if v == nil {
		return nil
	}
	
	list, ok := v.([]interface{})
	if !ok || len(list) == 0 {
		return nil
	}
	
	m, ok := list[0].(map[string]interface{})
	if !ok {
		return nil
	}
	
	return &VMAction{
		Type:      GetStringOrEmpty(m, "type"),
	}
}

// ParseSnapshotAction parses snapshot action from schema data
func ParseSnapshotAction(v interface{}) *SnapshotAction {
	if v == nil {
		return nil
	}
	
	list, ok := v.([]interface{})
	if !ok || len(list) == 0 {
		return nil
	}
	
	m, ok := list[0].(map[string]interface{})
	if !ok {
		return nil
	}
	
	action := &SnapshotAction{
		Type:      GetStringOrEmpty(m, "type"),
	}
	
	// Handle include_ram field
	if includeRamVal, exists := m["include_ram"]; exists && includeRamVal != nil {
		if boolVal, isBool := includeRamVal.(bool); isBool {
			action.IncludeRam = &boolVal
		}
	}
	
	return action
}

// ParseTemplateAction parses template action from schema data
func ParseTemplateAction(v interface{}) *TemplateAction {
	if v == nil {
		return nil
	}
	
	list, ok := v.([]interface{})
	if !ok || len(list) == 0 {
		return nil
	}
	
	m, ok := list[0].(map[string]interface{})
	if !ok {
		return nil
	}
	
	return &TemplateAction{
		Type:        GetStringOrEmpty(m, "type"),
		Name:        GetStringOrEmpty(m, "name"),
		Description: GetStringOrEmpty(m, "description"),
	}
}

// GetFirstBlockMap lấy map[string]any đầu tiên từ một block (list of map) trong resource data
func GetFirstBlockMap(d *schema.ResourceData, key string) map[string]any {
	raw := d.Get(key)
	if raw == nil {
		return nil
	}
	list, ok := raw.([]interface{})
	if !ok || len(list) == 0 {
		return nil
	}
	m, ok := list[0].(map[string]interface{})
	if !ok {
		return nil
	}
	result := make(map[string]any, len(m))
	for k, v := range m {
		result[k] = v
	}
	return result
}

