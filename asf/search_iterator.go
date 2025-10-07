package asf

import (
	"context"
	"net/url"
	"strconv"

	"github.com/example/go-asf/asf/model"
)

// ResultIterator provides streaming access to paginated search results.
type ResultIterator struct {
	client    *Client
	query     url.Values
	pageSize  int
	page      int
	index     int
	batch     []model.Product
	lastErr   error
	exhausted bool
}

func newResultIterator(client *Client, query url.Values, pageSize int) *ResultIterator {
	if pageSize <= 0 {
		pageSize = 100
	}
	return &ResultIterator{
		client:   client,
		query:    cloneValues(query),
		pageSize: pageSize,
		page:     1,
	}
}

// Next fetches the next product. It returns false when iteration is complete or an error occurred.
func (it *ResultIterator) Next(ctx context.Context) bool {
	if it.exhausted {
		return false
	}

	if it.index < len(it.batch) {
		it.index++
		return true
	}

	if it.lastErr != nil {
		return false
	}

	if err := it.loadNext(ctx); err != nil {
		it.lastErr = err
		return false
	}

	if len(it.batch) == 0 {
		it.exhausted = true
		return false
	}

	it.index = 1
	return true
}

// Product returns the current product. Call after Next returns true.
func (it *ResultIterator) Product() model.Product {
	if it.index == 0 || it.index > len(it.batch) {
		return model.Product{}
	}
	return it.batch[it.index-1]
}

// Err reports any error encountered during iteration.
func (it *ResultIterator) Err() error {
	return it.lastErr
}

func (it *ResultIterator) loadNext(ctx context.Context) error {
	it.query.Set("page", strconv.Itoa(it.page))
	it.query.Set("maxResults", strconv.Itoa(it.pageSize))

	resp, err := it.client.doSearchRequest(ctx, it.query)
	if err != nil {
		return err
	}

	it.batch = resp.Results
	it.index = 0
	it.page++
	if len(resp.Results) == 0 {
		it.exhausted = true
	}
	return nil
}

func cloneValues(v url.Values) url.Values {
	cp := make(url.Values, len(v))
	for k, vals := range v {
		dup := make([]string, len(vals))
		copy(dup, vals)
		cp[k] = dup
	}
	return cp
}
