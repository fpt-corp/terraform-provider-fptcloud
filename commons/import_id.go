package commons

import (
	"fmt"
	"regexp"
	"strings"
)

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func ParseVpcImportId(importId string, kind string) (vpcId string, id string, err error) {
	parts := strings.Split(importId, "/")
	if len(parts) != 4 || parts[0] != "vpc" || parts[2] != kind || !uuidPattern.MatchString(parts[1]) || !uuidPattern.MatchString(parts[3]) {
		return "", "", fmt.Errorf("invalid import id %q: expected vpc/<vpc_id>/%s/<%s_id>, where both ids must be UUIDs", importId, kind, kind)
	}
	return strings.ToLower(parts[1]), strings.ToLower(parts[3]), nil
}

func IsVpcImportName(importId string, kind string) bool {
	parts := strings.SplitN(importId, "/", 4)
	return len(parts) >= 3 && parts[0] == "vpc" && parts[2] == kind+"_name"
}

func ParseVpcImportName(importId string, kind string) (vpcId string, name string, err error) {
	parts := strings.SplitN(importId, "/", 4)
	if len(parts) != 4 || parts[0] != "vpc" || parts[2] != kind+"_name" || !uuidPattern.MatchString(parts[1]) || strings.TrimSpace(parts[3]) == "" {
		return "", "", fmt.Errorf("invalid import id %q: expected vpc/<vpc_id>/%s_name/<%s_name>, where vpc_id must be a UUID", importId, kind, kind)
	}
	return strings.ToLower(parts[1]), parts[3], nil
}
