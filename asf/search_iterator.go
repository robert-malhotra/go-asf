package asf

import (
	"context"
	"net/url"

	"github.com/example/go-asf/asf/model"
)

// ResultIterator provides streaming access to search results.
type ResultIterator struct {
	client  *Client
	query   url.Values
	fetched bool
	index   int
	batch   []model.Product
	lastErr error
}

func newResultIterator(client *Client, query url.Values, _ int) *ResultIterator {
	return &ResultIterator{
		client: client,
		query:  cloneValues(query),
	}
}

// Next fetches the next product. It returns false when iteration is complete or an error occurred.
func (it *ResultIterator) Next(ctx context.Context) bool {
	if it.lastErr != nil {
		return false
	}

	if it.index < len(it.batch) {
		it.index++
		return true
	}

	if it.fetched {
		return false
	}

	if err := it.load(ctx); err != nil {
		it.lastErr = err
		return false
	}

	if len(it.batch) == 0 {
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

func (it *ResultIterator) load(ctx context.Context) error {
	resp, err := it.client.doSearchRequest(ctx, it.query)
	if err != nil {
		return err
	}
	it.batch = resp.Results
	it.index = 0
	it.fetched = true
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
