package commons

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func pagesOf(pages [][]string, total int, calls *int) func(page int) ([]string, int, error) {
	return func(page int) ([]string, int, error) {
		*calls++
		if page > len(pages) {
			return []string{}, total, nil
		}
		return pages[page-1], total, nil
	}
}

func identity(s string) string { return s }

func TestListAllPages_StopsOnShortPage(t *testing.T) {
	calls := 0
	items, err := ListAllPages(2, pagesOf([][]string{{"a", "b"}, {"c"}}, 3, &calls), identity)
	assert.NoError(t, err)
	assert.Equal(t, []string{"a", "b", "c"}, items)
	assert.Equal(t, 2, calls)
}

func TestListAllPages_StopsWhenTotalReached(t *testing.T) {
	calls := 0
	items, err := ListAllPages(2, pagesOf([][]string{{"a", "b"}, {"c", "d"}}, 4, &calls), identity)
	assert.NoError(t, err)
	assert.Equal(t, []string{"a", "b", "c", "d"}, items)
	assert.Equal(t, 2, calls, "a full last page must not trigger an extra request")
}

func TestListAllPages_EmptyResult(t *testing.T) {
	calls := 0
	items, err := ListAllPages(ListPageSize, pagesOf(nil, 0, &calls), identity)
	assert.NoError(t, err)
	assert.Empty(t, items)
	assert.Equal(t, 1, calls)
}

func TestListAllPages_SkipsItemShiftedOntoNextPage(t *testing.T) {
	calls := 0
	items, err := ListAllPages(2, pagesOf([][]string{{"a", "b"}, {"b", "c"}}, 3, &calls), identity)
	assert.NoError(t, err)
	assert.Equal(t, []string{"a", "b", "c"}, items)
}

func TestListAllPages_ReturnsFetchError(t *testing.T) {
	_, err := ListAllPages(10, func(page int) ([]string, int, error) {
		return nil, 0, errors.New("boom")
	}, identity)
	assert.EqualError(t, err, "boom")
}

func TestListAllPages_RejectsOutOfRangePageSize(t *testing.T) {
	for _, size := range []int{0, ListPageSize + 1} {
		_, err := ListAllPages(size, func(page int) ([]string, int, error) {
			t.Fatal("no request may be sent for an invalid page size")
			return nil, 0, nil
		}, identity)
		assert.ErrorContains(t, err, "page_size")
	}
}
