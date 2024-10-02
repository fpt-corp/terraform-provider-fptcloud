package fptcloud_database

import (
	commons "terraform-provider-fptcloud/commons"
)

type databaseApiClient struct {
	*commons.Client
	edgeClient *commons.Client
}

func newDatabaseApiClient(c *commons.Client) (*databaseApiClient, error) {
	serviceToken := "eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJzb21lIjoicGF5bG9hZCJ9.Joh1R2dYzkRvDkqv3sygm5YyK8Gi4ShZqbhK2gxcs2U"
	edgeClient, err := commons.NewClientWithURL(
		serviceToken,
		c.BaseURL.String(),
		c.Region,
		c.TenantName,
	)

	if err != nil {
		return nil, err
	}

	return &databaseApiClient{
		Client:     edgeClient,
		edgeClient: edgeClient,
	}, nil
}
