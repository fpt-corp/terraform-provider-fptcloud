package commons

import "fmt"

// ListAllPages walks a paginated portal list endpoint and returns every item.
//
// fetchPage is called with page = 1, 2, ... and must return that page's items
// together with the total item count reported by the API. Paging stops at the
// first short page or once page*pageSize reaches the total, so collecting N
// items costs ceil(N/pageSize) requests. Items whose id has already been seen
// are skipped: the portal orders these lists by creation time, so a resource
// created while paging can shift an item onto the next page.
func ListAllPages[T any](pageSize int, fetchPage func(page int) ([]T, int, error), id func(T) string) ([]T, error) {
	if pageSize < 1 || pageSize > ListPageSize {
		return nil, fmt.Errorf("page_size must be between 1 and %d, got %d", ListPageSize, pageSize)
	}

	all := []T{}
	seen := map[string]bool{}

	for page := 1; ; page++ {
		items, total, err := fetchPage(page)
		if err != nil {
			return nil, err
		}

		for _, item := range items {
			key := id(item)
			if seen[key] {
				continue
			}
			seen[key] = true
			all = append(all, item)
		}

		if len(items) < pageSize || page*pageSize >= total {
			return all, nil
		}
	}
}
