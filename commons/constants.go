package commons

const (
	DefaultApiUrl = "https://console-api.fptcloud.com/api"

	// ListPageSize is the default and the maximum page size the provider uses
	// when walking the portal's paginated list endpoints (instances, security
	// groups, storages). Capping it keeps every request small, so fetching N
	// resources issues ceil(N/page_size) requests rather than one request per
	// resource or a single oversized request.
	ListPageSize = 100
)
