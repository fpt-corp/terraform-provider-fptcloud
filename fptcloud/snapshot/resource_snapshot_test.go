package fptcloud_snapshot_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	fptcloud_snapshot "terraform-provider-fptcloud/fptcloud/snapshot"

	common "terraform-provider-fptcloud/commons"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/assert"

	test_helper "terraform-provider-fptcloud/commons/test-helper"
)

func TestResourceSnapshot_SchemaIsValid(t *testing.T) {
	r := fptcloud_snapshot.ResourceSnapshot()

	assert.NoError(t, r.InternalValidate(nil, true))
}

// The snapshot cannot be renamed or re-pointed at another instance, so every
// configurable attribute must force a new resource and Update must not exist.
func TestResourceSnapshot_ConfigurableFieldsForceNew(t *testing.T) {
	r := fptcloud_snapshot.ResourceSnapshot()

	for _, field := range []string{"vpc_id", "instance_id", "name", "tag_ids", "include_ram"} {
		t.Run(field, func(t *testing.T) {
			s, ok := r.Schema[field]
			assert.True(t, ok, "field %s is missing", field)
			assert.True(t, s.ForceNew, "field %s must be ForceNew", field)
		})
	}

	assert.Nil(t, r.UpdateContext, "snapshots are immutable, there must be no Update")
}

func TestResourceSnapshot_RequiredAndComputedFields(t *testing.T) {
	r := fptcloud_snapshot.ResourceSnapshot()

	for _, field := range []string{"vpc_id", "instance_id"} {
		assert.True(t, r.Schema[field].Required, "field %s must be required", field)
	}

	// name is required on OpenStack and meaningless on VMware, so the schema
	// leaves it optional and the backend enforces it where it applies.
	assert.False(t, r.Schema["name"].Required)
	assert.True(t, r.Schema["name"].Optional)
	// include_ram is the mirror image: VMware only.
	assert.True(t, r.Schema["include_ram"].Optional)
	assert.True(t, r.Schema["include_ram"].ForceNew)

	for _, field := range []string{"status", "created_at", "created_by", "size_gb", "snapshot_type"} {
		s, ok := r.Schema[field]
		assert.True(t, ok, "field %s is missing", field)
		assert.True(t, s.Computed, "field %s must be computed", field)
		assert.False(t, s.Required, "field %s must not be required", field)
	}
}

// The platform has no description column, so exposing one would silently drop
// whatever the practitioner wrote.
func TestResourceSnapshot_DoesNotExposeUnsupportedFields(t *testing.T) {
	r := fptcloud_snapshot.ResourceSnapshot()

	_, hasDescription := r.Schema["description"]
	assert.False(t, hasDescription, "the platform does not store a snapshot description")

	// The name the platform mints from the requested one is an implementation
	// detail; the snapshot is addressed by id.
	_, hasInternalName := r.Schema["internal_name"]
	assert.False(t, hasInternalName, "the generated name is not part of the resource")
}

func TestResourceSnapshot_NameValidation(t *testing.T) {
	validate := fptcloud_snapshot.ResourceSnapshot().Schema["name"].ValidateFunc

	valid := []string{"before-upgrade", "snap_01", "release 1.2", "a"}
	for _, name := range valid {
		t.Run("valid/"+name, func(t *testing.T) {
			_, errs := validate(name, "name")
			assert.Empty(t, errs)
		})
	}

	invalid := map[string]string{
		"empty":      "",
		"whitespace": "   ",
		"too long":   strings.Repeat("a", fptcloud_snapshot.MaxNameLength+1),
		"slash":      "before/upgrade",
		"at sign":    "snap@01",
	}
	for label, name := range invalid {
		t.Run("invalid/"+label, func(t *testing.T) {
			_, errs := validate(name, "name")
			assert.NotEmpty(t, errs)
		})
	}
}

func TestResourceSnapshot_ImporterParsesCompositeId(t *testing.T) {
	r := fptcloud_snapshot.ResourceSnapshot()
	d := r.Data(nil)
	d.SetId("vpc/vpc-1/snapshot/snapshot-1")

	results, err := r.Importer.StateContext(context.Background(), d, nil)

	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "snapshot-1", results[0].Id())
	assert.Equal(t, "vpc-1", results[0].Get("vpc_id"))
}

func TestResourceSnapshot_ImporterRejectsBadId(t *testing.T) {
	r := fptcloud_snapshot.ResourceSnapshot()

	bad := []string{
		"snapshot-1",
		"vpc/vpc-1/certificate/snapshot-1",
		"vpc/vpc-1/snapshot",
		"vpc/vpc-1/snapshot/snapshot-1/extra",
		"",
	}
	for _, id := range bad {
		t.Run(id, func(t *testing.T) {
			d := r.Data(nil)
			d.SetId(id)

			results, err := r.Importer.StateContext(context.Background(), d, nil)

			assert.Error(t, err)
			assert.Nil(t, results)
		})
	}
}

func TestResourceSnapshot_IsRegisteredOnTheProvider(t *testing.T) {
	r, ok := test_helper.TestProvider.ResourcesMap["fptcloud_snapshot"]

	assert.True(t, ok, "fptcloud_snapshot is not registered on the provider")
	assert.NotNil(t, r)
}

func TestAccFptCloudSnapshot_basic(t *testing.T) {
	vpcId := os.Getenv("VPC_ID")
	instanceId := os.Getenv("INSTANCE_ID")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			test_helper.TestPreCheck(t)
			if vpcId == "" || instanceId == "" {
				t.Skip("VPC_ID and INSTANCE_ID must be set for this acceptance test")
			}
		},
		ProviderFactories: test_helper.TestProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSnapshotConfig(vpcId, instanceId),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("fptcloud_snapshot.test", "id"),
					resource.TestCheckResourceAttr("fptcloud_snapshot.test", "name", "tf-acc-snapshot"),
					resource.TestCheckResourceAttr("fptcloud_snapshot.test", "instance_id", instanceId),
					resource.TestCheckResourceAttr(
						"fptcloud_snapshot.test", "status",
						fptcloud_snapshot.StatusAvailable,
					),
					resource.TestCheckResourceAttrSet("fptcloud_snapshot.test", "created_at"),
				),
			},
			{
				// A second apply of the same config must be a no-op: the
				// snapshot is found by id and every attribute round-trips.
				Config:   testAccSnapshotConfig(vpcId, instanceId),
				PlanOnly: true,
			},
			{
				ResourceName:      "fptcloud_snapshot.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["fptcloud_snapshot.test"]
					if !ok {
						return "", fmt.Errorf("resource not found in state")
					}
					return fmt.Sprintf("vpc/%s/snapshot/%s", vpcId, rs.Primary.ID), nil
				},
			},
		},
	})
}

func testAccSnapshotConfig(vpcId, instanceId string) string {
	return fmt.Sprintf(`
resource "fptcloud_snapshot" "test" {
  vpc_id      = %q
  instance_id = %q
  name        = "tf-acc-snapshot"
}
`, vpcId, instanceId)
}

// The v2 API answers with one snapshot resource, so the fixtures are the ones
// declared in snapshot_service_test.go.

// Reading back the platform's mangled name would show a diff on a ForceNew
// field, so a second apply would destroy and recreate the snapshot. It must
// leave the configured name alone.
func TestResourceSnapshot_ReadKeepsTheConfiguredName(t *testing.T) {
	client, server, _ := common.NewClientForTesting(map[string]string{detailPath: settledSnapshot})
	defer server.Close()

	r := fptcloud_snapshot.ResourceSnapshot()
	d := schema.TestResourceDataRaw(t, r.Schema, map[string]interface{}{
		"vpc_id":      "vpc-1",
		"instance_id": "instance-1",
		"name":        "before-upgrade",
	})
	d.SetId("snapshot-1")

	diags := r.ReadContext(context.Background(), d, client)

	assert.False(t, diags.HasError(), "read reported: %v", diags)
	assert.Equal(t, "before-upgrade", d.Get("name"))
	assert.Equal(t, "ACTIVE", d.Get("status"))
	assert.Equal(t, 40.0, d.Get("size_gb"))
	assert.Equal(t, "instance-1", d.Get("instance_id"))
}

// After an import there is no configured name yet, so Read has to fill it in.
func TestResourceSnapshot_ReadRecoversTheNameAfterImport(t *testing.T) {
	client, server, _ := common.NewClientForTesting(map[string]string{detailPath: settledSnapshot})
	defer server.Close()

	r := fptcloud_snapshot.ResourceSnapshot()
	d := schema.TestResourceDataRaw(t, r.Schema, map[string]interface{}{"vpc_id": "vpc-1"})
	d.SetId("snapshot-1")

	diags := r.ReadContext(context.Background(), d, client)

	assert.False(t, diags.HasError(), "read reported: %v", diags)
	assert.Equal(t, "before-upgrade", d.Get("name"))
}

// A snapshot deleted outside Terraform must drop out of state so the next plan
// recreates it, rather than failing the refresh forever.
func TestResourceSnapshot_ReadClearsIdWhenSnapshotIsGone(t *testing.T) {
	client, server := newClientWithStatus(t, 404, `{"status": false, "error_code": "SNAPSHOT_NOT_FOUND"}`)
	defer server.Close()

	r := fptcloud_snapshot.ResourceSnapshot()
	d := schema.TestResourceDataRaw(t, r.Schema, map[string]interface{}{
		"vpc_id": "vpc-1",
		"name":   "before-upgrade",
	})
	d.SetId("snapshot-1")

	diags := r.ReadContext(context.Background(), d, client)

	assert.False(t, diags.HasError(), "read reported: %v", diags)
	assert.Equal(t, "", d.Id())
}

// Destroying something that is already gone has reached the desired state.
func TestResourceSnapshot_DeleteIsIdempotent(t *testing.T) {
	client, server := newClientWithStatus(t, 404, `{"status": false, "error_code": "SNAPSHOT_NOT_FOUND"}`)
	defer server.Close()

	r := fptcloud_snapshot.ResourceSnapshot()
	d := schema.TestResourceDataRaw(t, r.Schema, map[string]interface{}{"vpc_id": "vpc-1"})
	d.SetId("snapshot-1")

	diags := r.DeleteContext(context.Background(), d, client)

	assert.False(t, diags.HasError(), "delete reported: %v", diags)
	assert.Equal(t, "", d.Id())
}

// Read cannot reach the vpc-scoped endpoint without a vpc id, and must say so
// instead of silently dropping the resource from state.
func TestResourceSnapshot_ReadFailsWithoutVpcId(t *testing.T) {
	client, server, _ := common.NewClientForTesting(map[string]string{detailPath: settledSnapshot})
	defer server.Close()

	r := fptcloud_snapshot.ResourceSnapshot()
	d := schema.TestResourceDataRaw(t, r.Schema, map[string]interface{}{})
	d.SetId("snapshot-1")

	diags := r.ReadContext(context.Background(), d, client)

	assert.True(t, diags.HasError())
	assert.Equal(t, "snapshot-1", d.Id())
}

// A create the platform refused must not leave anything in state.
func TestResourceSnapshot_CreateLeavesNoIdWhenPlatformRefuses(t *testing.T) {
	client, server := newClientWithStatus(t, 429, `{"status": false, "error_code": "QUOTA_ERROR", "message": "CREATE_FAIL"}`)
	defer server.Close()

	r := fptcloud_snapshot.ResourceSnapshot()
	d := schema.TestResourceDataRaw(t, r.Schema, map[string]interface{}{
		"vpc_id":      "vpc-1",
		"instance_id": "instance-1",
		"name":        "before-upgrade",
	})

	diags := r.CreateContext(context.Background(), d, client)

	assert.True(t, diags.HasError())
	assert.Contains(t, diags[0].Summary, "QUOTA_ERROR")
	assert.Equal(t, "", d.Id())
}

// The happy path: the snapshot settles, its id becomes the resource id, and the
// configured name survives.
func TestResourceSnapshot_CreateSucceeds(t *testing.T) {
	client, server, _ := common.NewClientForTesting(map[string]string{
		// The detail path is a prefix match on the collection path, so the
		// order the test server checks them in does not matter here: both
		// return the settled snapshot once creation has been accepted.
		collectionPath: settledSnapshot,
	})
	defer server.Close()

	r := fptcloud_snapshot.ResourceSnapshot()
	d := schema.TestResourceDataRaw(t, r.Schema, map[string]interface{}{
		"vpc_id":      "vpc-1",
		"instance_id": "instance-1",
		"name":        "before-upgrade",
	})

	diags := r.CreateContext(context.Background(), d, client)

	assert.False(t, diags.HasError(), "create reported: %v", diags)
	assert.Equal(t, "snapshot-1", d.Id())
	assert.Equal(t, "before-upgrade", d.Get("name"))
	assert.Equal(t, "ACTIVE", d.Get("status"))
}

// A snapshot that was created but never settles must still be recorded, or the
// next apply would ask the platform for a second snapshot under a name it has
// already taken.
func TestResourceSnapshot_CreateRecordsIdWhenSnapshotFailsToSettle(t *testing.T) {
	client, server, _ := common.NewClientForTesting(map[string]string{
		collectionPath: `{"status": true, "data": {"id": "snapshot-1", "status": "ERROR"}}`,
	})
	defer server.Close()

	r := fptcloud_snapshot.ResourceSnapshot()
	d := schema.TestResourceDataRaw(t, r.Schema, map[string]interface{}{
		"vpc_id":      "vpc-1",
		"instance_id": "instance-1",
		"name":        "before-upgrade",
	})

	diags := r.CreateContext(context.Background(), d, client)

	assert.True(t, diags.HasError())
	assert.Equal(t, "snapshot-1", d.Id())
}

// OpenStack reports the Glance image status uppercased, so a finished snapshot
// is ACTIVE, never AVAILABLE. Targeting AVAILABLE alone made every successful
// create fail with "unexpected state 'ACTIVE'".
func TestResourceSnapshot_CreateSettlesOnActive(t *testing.T) {
	client, server, _ := common.NewClientForTesting(map[string]string{
		collectionPath: `{"status": true, "data": {"id": "snapshot-1", "name": "before-upgrade", "status": "ACTIVE"}}`,
	})
	defer server.Close()

	r := fptcloud_snapshot.ResourceSnapshot()
	d := schema.TestResourceDataRaw(t, r.Schema, map[string]interface{}{
		"vpc_id":      "vpc-1",
		"instance_id": "instance-1",
		"name":        "before-upgrade",
	})

	diags := r.CreateContext(context.Background(), d, client)

	assert.False(t, diags.HasError(), "create reported: %v", diags)
	assert.Equal(t, "snapshot-1", d.Id())
	assert.Equal(t, "ACTIVE", d.Get("status"))
}

// VMware reports its own vocabulary, so the wait has to accept that too.
func TestResourceSnapshot_CreateSettlesOnAvailable(t *testing.T) {
	client, server, _ := common.NewClientForTesting(map[string]string{
		collectionPath: `{"status": true, "data": {"id": "snapshot-1", "name": "before-upgrade", "status": "AVAILABLE"}}`,
	})
	defer server.Close()

	r := fptcloud_snapshot.ResourceSnapshot()
	d := schema.TestResourceDataRaw(t, r.Schema, map[string]interface{}{
		"vpc_id":      "vpc-1",
		"instance_id": "instance-1",
		"name":        "before-upgrade",
	})

	diags := r.CreateContext(context.Background(), d, client)

	assert.False(t, diags.HasError(), "create reported: %v", diags)
	assert.Equal(t, "AVAILABLE", d.Get("status"))
}

// Every state the platform can report as a failure must abort the wait rather
// than be polled until the timeout.
func TestResourceSnapshot_CreateAbortsOnEveryFailedStatus(t *testing.T) {
	for _, status := range []string{"ERROR", "ERROR_DELETING", "KILLED", "DEACTIVATED"} {
		t.Run(status, func(t *testing.T) {
			client, server, _ := common.NewClientForTesting(map[string]string{
				collectionPath: `{"status": true, "data": {"id": "snapshot-1", "status": "` + status + `"}}`,
			})
			defer server.Close()

			r := fptcloud_snapshot.ResourceSnapshot()
			d := schema.TestResourceDataRaw(t, r.Schema, map[string]interface{}{
				"vpc_id":      "vpc-1",
				"instance_id": "instance-1",
				"name":        "before-upgrade",
			})

			diags := r.CreateContext(context.Background(), d, client)

			assert.True(t, diags.HasError())
			assert.Contains(t, diags[0].Summary, status)
			// Still recorded, so it can be destroyed rather than orphaned.
			assert.Equal(t, "snapshot-1", d.Id())
		})
	}
}

func TestSnapshotStatusSets(t *testing.T) {
	// A finished snapshot must never be treated as still in flight, and the
	// two vocabularies must not overlap with the failure set.
	assert.Contains(t, fptcloud_snapshot.SettledStatuses, "ACTIVE")
	assert.Contains(t, fptcloud_snapshot.SettledStatuses, "AVAILABLE")

	for _, status := range fptcloud_snapshot.SettledStatuses {
		assert.False(t, fptcloud_snapshot.IsFailed(status), "%s must not count as failed", status)
		assert.NotContains(t, fptcloud_snapshot.CreatingStatuses, status)
	}

	for _, status := range fptcloud_snapshot.CreatingStatuses {
		assert.False(t, fptcloud_snapshot.IsFailed(status), "%s must not count as failed", status)
	}

	// A snapshot on its way out is polled, not aborted on.
	for _, status := range fptcloud_snapshot.DeletingStatuses {
		assert.False(t, fptcloud_snapshot.IsFailed(status), "%s must not count as failed", status)
	}

	assert.True(t, fptcloud_snapshot.IsFailed("ERROR"))
	assert.True(t, fptcloud_snapshot.IsFailed("ERROR_DELETING"))
	assert.True(t, fptcloud_snapshot.IsFailed("KILLED"))
	assert.False(t, fptcloud_snapshot.IsFailed("CREATING"))
	assert.False(t, fptcloud_snapshot.IsFailed(""))
}

// A snapshot whose deletion failed must surface, not be polled until timeout.
func TestResourceSnapshot_DeleteAbortsWhenDeletionFails(t *testing.T) {
	client, server, _ := common.NewClientForTesting(map[string]string{
		detailPath: `{"status": true, "data": {"id": "snapshot-1", "status": "ERROR_DELETING"}}`,
	})
	defer server.Close()

	r := fptcloud_snapshot.ResourceSnapshot()
	d := schema.TestResourceDataRaw(t, r.Schema, map[string]interface{}{"vpc_id": "vpc-1"})
	d.SetId("snapshot-1")

	diags := r.DeleteContext(context.Background(), d, client)

	assert.True(t, diags.HasError())
	assert.Contains(t, diags[0].Summary, "ERROR_DELETING")
	// Kept in state: the snapshot is still there and still costs quota.
	assert.Equal(t, "snapshot-1", d.Id())
}

// VMware stores no snapshot name, so a configuration that omits it must still
// create — the same HCL has to work whichever platform serves the vpc.
func TestResourceSnapshot_CreateWorksWithoutAName(t *testing.T) {
	client, server, _ := common.NewClientForTesting(map[string]string{
		collectionPath: `{"status": true, "data": {"id": "snapshot-1", "name": null, "status": "AVAILABLE"}}`,
	})
	defer server.Close()

	r := fptcloud_snapshot.ResourceSnapshot()
	d := schema.TestResourceDataRaw(t, r.Schema, map[string]interface{}{
		"vpc_id":      "vpc-1",
		"instance_id": "instance-1",
		"include_ram": true,
	})

	diags := r.CreateContext(context.Background(), d, client)

	assert.False(t, diags.HasError(), "create reported: %v", diags)
	assert.Equal(t, "snapshot-1", d.Id())
}

// Tags come back from the API so drift on them is visible, rather than tag_ids
// being a write-only field nobody can verify.
func TestResourceSnapshot_ReadSurfacesTags(t *testing.T) {
	client, server, _ := common.NewClientForTesting(map[string]string{
		detailPath: `{"status": true, "data": {
			"id": "snapshot-1",
			"name": "before-upgrade",
			"status": "ACTIVE",
			"tags": [{"id": "tag-1", "key": "env", "value": "prod"}]
		}}`,
	})
	defer server.Close()

	r := fptcloud_snapshot.ResourceSnapshot()
	d := schema.TestResourceDataRaw(t, r.Schema, map[string]interface{}{
		"vpc_id": "vpc-1",
		"name":   "before-upgrade",
	})
	d.SetId("snapshot-1")

	diags := r.ReadContext(context.Background(), d, client)

	assert.False(t, diags.HasError(), "read reported: %v", diags)
	tags := d.Get("tags").([]interface{})
	assert.Len(t, tags, 1)
	assert.Equal(t, "env", tags[0].(map[string]interface{})["key"])
	assert.Equal(t, "prod", tags[0].(map[string]interface{})["value"])
}

func TestResourceSnapshot_ReadHandlesNoTags(t *testing.T) {
	client, server, _ := common.NewClientForTesting(map[string]string{
		detailPath: `{"status": true, "data": {"id": "snapshot-1", "status": "ACTIVE"}}`,
	})
	defer server.Close()

	r := fptcloud_snapshot.ResourceSnapshot()
	d := schema.TestResourceDataRaw(t, r.Schema, map[string]interface{}{"vpc_id": "vpc-1"})
	d.SetId("snapshot-1")

	diags := r.ReadContext(context.Background(), d, client)

	assert.False(t, diags.HasError(), "read reported: %v", diags)
	assert.Empty(t, d.Get("tags").([]interface{}))
}

// VMware marks the row deleted rather than removing it. If a destroy ever sees
// that state it must read as "on its way out", not as an unexpected state that
// aborts the wait.
func TestResourceSnapshot_DeleteToleratesTheDeletedState(t *testing.T) {
	assert.Contains(t, fptcloud_snapshot.DeletingStatuses, "DELETED")
	assert.False(t, fptcloud_snapshot.IsFailed("DELETED"))
	assert.NotContains(t, fptcloud_snapshot.SettledStatuses, "DELETED")
}

// VMware holds one snapshot per instance and taking another replaces it, so a
// create that replaced one is a normal success — the resource simply owns the
// snapshot the instance now holds.
func TestResourceSnapshot_CreateAcceptsAVmwareReplacement(t *testing.T) {
	client, server, _ := common.NewClientForTesting(map[string]string{
		collectionPath: `{"status": true, "data": {
			"id": "snapshot-1", "status": "AVAILABLE", "replaced": true
		}}`,
	})
	defer server.Close()

	r := fptcloud_snapshot.ResourceSnapshot()
	d := schema.TestResourceDataRaw(t, r.Schema, map[string]interface{}{
		"vpc_id":      "vpc-1",
		"instance_id": "instance-1",
	})

	diags := r.CreateContext(context.Background(), d, client)

	assert.False(t, diags.HasError(), "create reported: %v", diags)
	assert.Equal(t, "snapshot-1", d.Id())
	assert.Equal(t, "AVAILABLE", d.Get("status"))
}

// A replacement starts in CREATING and has to be waited out like any other, or
// the resource would report a snapshot that is still being taken as settled.
func TestResourceSnapshot_CreateWaitsOutAReplacement(t *testing.T) {
	// The collection path is a prefix of the detail path, so the create and the
	// poll have to be told apart by method rather than by url.
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.Method == http.MethodPost {
			_, _ = rw.Write([]byte(`{"status": true, "data": {"id": "snapshot-1", "status": "CREATING", "replaced": true}}`))
			return
		}
		_, _ = rw.Write([]byte(`{"status": true, "data": {"id": "snapshot-1", "status": "AVAILABLE"}}`))
	}))
	defer server.Close()

	client, err := common.NewClientForTestingWithServer(server)
	assert.NoError(t, err)

	r := fptcloud_snapshot.ResourceSnapshot()
	d := schema.TestResourceDataRaw(t, r.Schema, map[string]interface{}{
		"vpc_id":      "vpc-1",
		"instance_id": "instance-1",
	})

	diags := r.CreateContext(context.Background(), d, client)

	assert.False(t, diags.HasError(), "create reported: %v", diags)
	assert.Equal(t, "AVAILABLE", d.Get("status"))
}

// On OpenStack a snapshot is a Glance image, which does not record the instance
// it came from, and the sync that reconciles the record clears vm_id. Reading
// that empty value back over a known one would look like a change on a ForceNew
// field and have Terraform replace the snapshot on every plan.
func TestResourceSnapshot_ReadKeepsInstanceIdWhenApiForgetsIt(t *testing.T) {
	client, server, _ := common.NewClientForTesting(map[string]string{
		detailPath: `{"status": true, "data": {"id": "snapshot-1", "vm_id": "", "status": "ACTIVE"}}`,
	})
	defer server.Close()

	r := fptcloud_snapshot.ResourceSnapshot()
	d := schema.TestResourceDataRaw(t, r.Schema, map[string]interface{}{
		"vpc_id":      "vpc-1",
		"instance_id": "instance-1",
		"name":        "before-upgrade",
	})
	d.SetId("snapshot-1")

	diags := r.ReadContext(context.Background(), d, client)

	assert.False(t, diags.HasError(), "read reported: %v", diags)
	assert.Equal(t, "instance-1", d.Get("instance_id"))
}

// When the platform does report it, the value is taken as authoritative.
func TestResourceSnapshot_ReadTakesInstanceIdWhenApiReportsIt(t *testing.T) {
	client, server, _ := common.NewClientForTesting(map[string]string{
		detailPath: `{"status": true, "data": {"id": "snapshot-1", "vm_id": "instance-9", "status": "ACTIVE"}}`,
	})
	defer server.Close()

	r := fptcloud_snapshot.ResourceSnapshot()
	d := schema.TestResourceDataRaw(t, r.Schema, map[string]interface{}{"vpc_id": "vpc-1"})
	d.SetId("snapshot-1")

	diags := r.ReadContext(context.Background(), d, client)

	assert.False(t, diags.HasError(), "read reported: %v", diags)
	assert.Equal(t, "instance-9", d.Get("instance_id"))
}
