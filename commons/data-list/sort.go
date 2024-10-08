package data_list

import (
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"terraform-provider-fptcloud/commons/utils"
)

var (
	sortKeys = []string{"asc", "desc"}
)

type commonSort struct {
	key       string
	direction string
}

func sortSchema(resultAttributeName string, allowedKeys []string) *schema.Schema {
	return &schema.Schema{
		Type: schema.TypeList,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"key": {
					Type:         schema.TypeString,
					Required:     true,
					ValidateFunc: validation.StringInSlice(allowedKeys, false),
					Description:  fmt.Sprintf("Sort %s by this key. This may be one of %s.", resultAttributeName, utils.GetCommaSeparatedAllowedKeys(allowedKeys)),
				},
				"direction": {
					Type:         schema.TypeString,
					Optional:     true,
					ValidateFunc: validation.StringInSlice(sortKeys, false),
					Description:  "The sort direction. This may be either `asc` or `desc`.",
				},
			},
		},
		Optional:    true,
		Description: "One or more key/direction pairs on which to sort results",
	}
}

func expandSorts(rawSorts []interface{}) []commonSort {
	expandedSorts := make([]commonSort, len(rawSorts))
	for i, rawSort := range rawSorts {
		f := rawSort.(map[string]interface{})

		expandedSort := commonSort{
			key:       f["key"].(string),
			direction: f["direction"].(string),
		}

		expandedSorts[i] = expandedSort
	}
	return expandedSorts
}

func applySorts(recordSchema map[string]*schema.Schema, records []map[string]interface{}, sorts []commonSort) []map[string]interface{} {
	sort.Slice(records, func(_i, _j int) bool {
		for _, s := range sorts {
			// Handle multiple sorts by applying them in order
			i := _i
			j := _j
			if strings.EqualFold(s.direction, "desc") {
				// If the direction is desc, reverse index to compare
				i = _j
				j = _i
			}

			value1 := records[i]
			value2 := records[j]
			cmp := compareValues(recordSchema[s.key], value1[s.key], value2[s.key])
			if cmp != 0 {
				return cmp < 0
			}
		}

		return true
	})

	return records
}
