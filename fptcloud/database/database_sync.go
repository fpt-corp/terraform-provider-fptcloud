package fptcloud_database

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	common "terraform-provider-fptcloud/commons"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const (
	// syncWaitTimeout is how long a read is willing to wait for the sync calls. What
	// matters to the read is that the sync has been triggered, not that it has finished
	syncWaitTimeout = 3 * time.Second

	// syncCallTimeout bounds a single call so that a request nobody waits for any more
	// cannot stay around forever
	syncCallTimeout = 60 * time.Second
)

// syncVpcInfrastructure pulls the VM and storage state of a VPC from the infrastructure
// into BSS. The cluster detail endpoint syncs itself from BSS, but BSS is never filled
// automatically, so without these calls a freshly provisioned cluster keeps coming back
// without its node information.
//
// The three calls are fired in parallel and are best effort. After syncWaitTimeout the
// read carries on without them: they keep running in the background so BSS still gets
// filled, and their failures only end up in the log.
func syncVpcInfrastructure(ctx context.Context, client *common.Client, vpcId string) {
	if client == nil || vpcId == "" {
		return
	}

	paths := []string{
		common.ApiPath.VpcSyncInstances(vpcId),
		common.ApiPath.VpcSyncStorages(vpcId),
		common.ApiPath.VpcSyncStoragesV2(vpcId),
	}

	// Detached from the read: we stop waiting after syncWaitTimeout, but the calls
	// themselves have to survive that, otherwise nothing would ever reach BSS
	syncCtx := context.WithoutCancel(ctx)

	var wg sync.WaitGroup
	for _, path := range paths {
		wg.Add(1)
		go func(path string) {
			defer wg.Done()

			tflog.Debug(syncCtx, "Syncing VPC infrastructure, calling path "+path)
			if err := sendSyncRequest(syncCtx, client, path); err != nil {
				tflog.Warn(syncCtx, fmt.Sprintf("Failed syncing VPC infrastructure through %s: %v", path, err))
				return
			}
			tflog.Debug(syncCtx, "Synced VPC infrastructure through "+path)
		}(path)
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		tflog.Debug(ctx, "Finished syncing the infrastructure of VPC "+vpcId)
	case <-time.After(syncWaitTimeout):
		tflog.Info(ctx, "The infrastructure of VPC "+vpcId+" is still syncing, carrying on with the read")
	}
}

// sendSyncRequest posts to a sync endpoint under its own deadline
func sendSyncRequest(ctx context.Context, client *common.Client, path string) error {
	requestCtx, cancel := context.WithTimeout(ctx, syncCallTimeout)
	defer cancel()

	u := client.PrepareClientURL(path)
	req, err := http.NewRequestWithContext(requestCtx, "POST", u.String(), bytes.NewBufferString("{}"))
	if err != nil {
		return err
	}

	_, err = client.SendRequest(req)
	return err
}
