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
	syncWaitTimeout = 3 * time.Second

	syncCallTimeout = 60 * time.Second
)

func syncVpcInfrastructure(ctx context.Context, client *common.Client, vpcId string) {
	if client == nil || vpcId == "" {
		return
	}

	paths := []string{
		common.ApiPath.VpcSyncInstances(vpcId),
		common.ApiPath.VpcSyncStorages(vpcId),
		common.ApiPath.VpcSyncStoragesV2(vpcId),
	}

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
