package asf_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/robert-malhotra/go-asf/pkg/asf"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLiveSearchEquivalentToResponseJSON(t *testing.T) {
	// This test runs against the live API to validate the query from asf_response.json
	// It does NOT require credentials and should always run.

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	startTime, err := time.Parse(time.RFC3339, "2024-01-01T00:00:00Z")
	require.NoError(t, err)
	endTime, err := time.Parse(time.RFC3339, "2025-01-31T23:59:59Z")
	require.NoError(t, err)

	opts := asf.SearchOptions{
		Platforms:       []asf.Platform{asf.PlatformSentinel1},
		ProcessingLevel: []asf.ProcessingLevel{asf.ProcessingLevelSLC},
		Start:           startTime,
		End:             endTime,
		IntersectsWith:  "POLYGON ((-64.8 32.3, -65.5 18.3, -80.3 25.2, -64.8 32.3))",
	}

	client := asf.NewClient()
	products, err := client.Search(ctx, opts)
	require.NoError(t, err)

	assert.Greater(t, len(products), 1)

	token := os.Getenv("ASF_TOKEN")
	if token == "" {
		t.Skip("Skipping live download test: ASF_TOKEN not set")
	}

	client = asf.NewClient(asf.WithAuthToken(token))
	client.Download(t.Context(), t.TempDir(), products[0])
}
