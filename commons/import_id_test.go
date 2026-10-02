package commons

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	testVpcId      = "11111111-1111-4111-8111-111111111111"
	testResourceId = "aaaaaaaa-0000-4000-8000-00000000000a"
)

func TestParseVpcImportId_ReturnsVpcAndResourceIds(t *testing.T) {
	vpcId, id, err := ParseVpcImportId("vpc/"+testVpcId+"/instance/"+testResourceId, "instance")
	assert.NoError(t, err)
	assert.Equal(t, testVpcId, vpcId)
	assert.Equal(t, testResourceId, id)
}

func TestParseVpcImportId_LowercasesUpperCaseUuids(t *testing.T) {
	vpcId, id, err := ParseVpcImportId("vpc/11111111-1111-4111-8111-11111111111A/storage/AAAAAAAA-0000-4000-8000-00000000000A", "storage")
	assert.NoError(t, err)
	assert.Equal(t, "11111111-1111-4111-8111-11111111111a", vpcId)
	assert.Equal(t, "aaaaaaaa-0000-4000-8000-00000000000a", id)
}

func TestParseVpcImportId_RejectsMalformedIds(t *testing.T) {
	for _, importId := range []string{
		"",
		testResourceId,
		testVpcId + "/" + testResourceId,
		"vpc/" + testVpcId + "/storage/" + testResourceId,
		"vpc/" + testVpcId + "/instance/" + testResourceId + "/extra",
		"vpcs/" + testVpcId + "/instance/" + testResourceId,
		"vpc/" + testVpcId + "/instance/",
		"vpc/not-a-uuid/instance/" + testResourceId,
		"vpc/" + testVpcId + "/instance/vm-name",
		" vpc/" + testVpcId + "/instance/" + testResourceId,
	} {
		_, _, err := ParseVpcImportId(importId, "instance")
		if assert.Error(t, err, importId) {
			assert.Contains(t, err.Error(), "expected vpc/<vpc_id>/instance/<instance_id>")
			assert.Contains(t, err.Error(), "must be UUIDs")
		}
	}
}

func TestIsVpcImportName_OnlyMatchesTheNameForm(t *testing.T) {
	assert.True(t, IsVpcImportName("vpc/"+testVpcId+"/instance_name/web-x7k2p", "instance"))
	assert.True(t, IsVpcImportName("vpc/not-a-uuid/instance_name/", "instance"))
	assert.False(t, IsVpcImportName("vpc/"+testVpcId+"/instance/"+testResourceId, "instance"))
	assert.False(t, IsVpcImportName("vpc/"+testVpcId+"/instance/web-x7k2p", "instance"))
	assert.False(t, IsVpcImportName("vpc/"+testVpcId+"/storage_name/disk-1", "instance"))
	assert.False(t, IsVpcImportName(testResourceId, "instance"))
}

func TestParseVpcImportName_ReturnsVpcIdAndNameAsWritten(t *testing.T) {
	vpcId, name, err := ParseVpcImportName("vpc/11111111-1111-4111-8111-11111111111A/instance_name/Web-X7k2p", "instance")
	assert.NoError(t, err)
	assert.Equal(t, "11111111-1111-4111-8111-11111111111a", vpcId)
	assert.Equal(t, "Web-X7k2p", name)
}

func TestParseVpcImportName_KeepsSlashesInTheName(t *testing.T) {
	_, name, err := ParseVpcImportName("vpc/"+testVpcId+"/instance_name/team/web 01", "instance")
	assert.NoError(t, err)
	assert.Equal(t, "team/web 01", name)
}

func TestParseVpcImportName_RejectsMalformedIds(t *testing.T) {
	for _, importId := range []string{
		"vpc/" + testVpcId + "/instance_name/",
		"vpc/" + testVpcId + "/instance_name/   ",
		"vpc/" + testVpcId + "/instance_name",
		"vpc/not-a-uuid/instance_name/web",
		"vpcs/" + testVpcId + "/instance_name/web",
		"vpc/" + testVpcId + "/storage_name/web",
	} {
		_, _, err := ParseVpcImportName(importId, "instance")
		if assert.Error(t, err, importId) {
			assert.Contains(t, err.Error(), "expected vpc/<vpc_id>/instance_name/<instance_name>")
		}
	}
}
