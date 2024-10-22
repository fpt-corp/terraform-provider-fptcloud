package fptcloud_mfke

import (
	diag2 "github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"testing"
)

var (
	successVmw = &managedKubernetesEngine{
		NetworkID: types.StringValue(""),
		Pools: []*managedKubernetesEnginePool{
			{
				WorkerPoolID: types.StringValue("worker-1"),
				NetworkID:    types.StringValue("6436b770-8fd3-44d4-80ea-77e3e05b502f"),
			},
			{
				WorkerPoolID: types.StringValue("worker-2"),
				NetworkID:    types.StringValue("f9ff3950-a546-46ba-9ed8-625661245b1f"),
			},
		},
	}

	successOsp = &managedKubernetesEngine{
		NetworkID: types.StringValue("6436b770-8fd3-44d4-80ea-77e3e05b502f"),
		Pools: []*managedKubernetesEnginePool{
			{
				WorkerPoolID: types.StringValue("worker-1"),
				NetworkID:    types.StringValue("6436b770-8fd3-44d4-80ea-77e3e05b502f"),
			},
			{
				WorkerPoolID: types.StringValue("worker-2"),
				NetworkID:    types.StringValue("6436b770-8fd3-44d4-80ea-77e3e05b502f"),
			},
		},
	}
	poolDupe = []*managedKubernetesEnginePool{
		{WorkerPoolID: types.StringValue("worker-1")},
		{WorkerPoolID: types.StringValue("worker-1")},
	}
)

func TestValidate(t *testing.T) {
	var err *diag2.ErrorDiagnostic
	err = validateNetwork(successOsp, "osp")
	assert.Nil(t, err)

	err = validateNetwork(successVmw, "vmw")
	assert.Nil(t, err)

	err = validateNetwork(successVmw, "osp")
	assert.NotNil(t, err)

	err = validateNetwork(successOsp, "vmw")
	assert.NotNil(t, err)
}

func TestValidatePool(t *testing.T) {
	var diag *diag2.ErrorDiagnostic
	diag = validatePool([]*managedKubernetesEnginePool{})
	assert.NotNil(t, diag)

	diag = validatePool(nil)
	assert.NotNil(t, diag)

	diag = validatePool(poolDupe)
	assert.NotNil(t, diag)

	diag = validatePool(successVmw.Pools)
	assert.Nil(t, diag)
}

func TestValidatePoolName(t *testing.T) {
	names, err := validatePoolNames(successVmw.Pools)
	assert.NoError(t, err)
	assert.Len(t, names, 2)
	assert.Equal(t, "worker-1", names[0])
	assert.Equal(t, "worker-2", names[1])

	_, err = validatePoolNames(poolDupe)
	assert.Error(t, err)
}
