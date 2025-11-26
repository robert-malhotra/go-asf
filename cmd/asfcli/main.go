package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"net/http"
	"net/http/cookiejar"

	"github.com/urfave/cli/v3"

	"github.com/example/go-asf/pkg/asf"
)

func main() {
	root := &cli.Command{
		Name:    "asfcli",
		Usage:   "Search and download products from the Alaska Satellite Facility (ASF) API",
		Version: "0.1.0",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "token",
				Usage:   "Provide a bearer token for authenticated requests",
				Sources: cli.EnvVars("ASF_TOKEN"),
			},
		},
		Commands: []*cli.Command{
			newSearchCommand(),
		},
	}

	if err := root.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}

func newSearchCommand() *cli.Command {
	return &cli.Command{
		Name:  "search",
		Usage: "Execute a search against the ASF API",
		Flags: []cli.Flag{
			&cli.StringSliceFlag{
				Name:    "platform",
				Usage:   "Filter by platform (repeatable)",
				Aliases: []string{"p"},
			},
			&cli.StringSliceFlag{
				Name:    "beam-mode",
				Usage:   "Filter by beam mode (repeatable)",
				Aliases: []string{"b"},
			},
			&cli.StringSliceFlag{
				Name:  "polarization",
				Usage: "Filter by polarization (repeatable)",
			},
			&cli.StringSliceFlag{
				Name:  "product-type",
				Usage: "Filter by product type (repeatable)",
			},
			&cli.StringSliceFlag{
				Name:  "collection",
				Usage: "Filter by collection name (repeatable)",
			},
			&cli.StringSliceFlag{
				Name:  "processing-level",
				Usage: "Filter by processing level (repeatable)",
			},
			&cli.StringSliceFlag{
				Name:  "look-direction",
				Usage: "Filter by look direction (repeatable)",
			},
			&cli.StringFlag{
				Name:  "relative-orbit",
				Usage: "Filter by relative orbit",
			},
			&cli.StringFlag{
				Name:  "flight-direction",
				Usage: "Filter by flight direction (ASCENDING or DESCENDING)",
			},
			&cli.StringFlag{
				Name:  "intersects",
				Usage: "WKT or GeoJSON geometry for intersectsWith filter",
			},
			&cli.StringSliceFlag{
				Name:    "granule",
				Usage:   "Filter by specific granule IDs (repeatable)",
				Aliases: []string{"g"},
			},
			&cli.StringFlag{
				Name:  "start",
				Usage: "Start time (RFC3339)",
			},
			&cli.StringFlag{
				Name:  "end",
				Usage: "End time (RFC3339)",
			},
			&cli.IntFlag{
				Name:  "max-results",
				Usage: "Maximum number of results to return",
			},
			&cli.StringFlag{
				Name:  "output",
				Usage: "Output format (text or json)",
				Value: "text",
			},
			&cli.StringFlag{
				Name:  "download-dir",
				Usage: "Download all matching products to the specified directory",
			},
		},
		Action: executeSearch,
	}
}

func executeSearch(ctx context.Context, cmd *cli.Command) error {
	client := buildClient(cmd)

	start, err := parseTimeFlag(cmd, "start")
	if err != nil {
		return err
	}
	end, err := parseTimeFlag(cmd, "end")
	if err != nil {
		return err
	}

	opts := asf.SearchOptions{
		Platforms:       convertSlice[asf.Platform](cmd.StringSlice("platform")),
		BeamModes:       convertSlice[asf.BeamMode](cmd.StringSlice("beam-mode")),
		Polarizations:   convertSlice[asf.Polarization](cmd.StringSlice("polarization")),
		ProductTypes:    convertSlice[asf.ProductType](cmd.StringSlice("product-type")),
		Collections:     convertSlice[asf.CollectionName](cmd.StringSlice("collection")),
		ProcessingLevel: convertSlice[asf.ProcessingLevel](cmd.StringSlice("processing-level")),
		LookDirections:  convertSlice[asf.LookDirection](cmd.StringSlice("look-direction")),
		RelativeOrbit:   strings.TrimSpace(cmd.String("relative-orbit")),
		FlightDirection: asf.FlightDirection(strings.TrimSpace(cmd.String("flight-direction"))),
		IntersectsWith:  strings.TrimSpace(cmd.String("intersects")),
		GranuleIDs:      trimStrings(cmd.StringSlice("granule")),
		Start:           start,
		End:             end,
		MaxResults:      cmd.Int("max-results"),
	}

	products, err := client.Search(ctx, opts)
	if err != nil {
		return fmt.Errorf("search: %w", err)
	}

	if len(products) == 0 {
		fmt.Fprintln(os.Stdout, "No products found.")
		return nil
	}

	switch output := strings.ToLower(strings.TrimSpace(cmd.String("output"))); output {
	case "json":
		if err := writeJSON(os.Stdout, products); err != nil {
			return err
		}
	case "text":
		printProductsTable(os.Stdout, products)
	default:
		return fmt.Errorf("unsupported output format %q", output)
	}

	downloadDir := strings.TrimSpace(cmd.String("download-dir"))
	if downloadDir == "" {
		return nil
	}

	fmt.Fprintf(os.Stderr, "Downloading %d product(s) to %s...\n", len(products), downloadDir)
	if err := client.Download(ctx, downloadDir, products...); err != nil {
		return fmt.Errorf("download: %w", err)
	}
	return nil
}

func buildClient(cmd *cli.Command) *asf.Client {
	var opts []asf.Option
	root := cmd.Root()
	if baseURL := strings.TrimSpace(root.String("base-url")); baseURL != "" {
		opts = append(opts, asf.WithBaseURL(baseURL))
	}
	if token := strings.TrimSpace(root.String("token")); token != "" {
		opts = append(opts, asf.WithAuthToken(token))
	}
	timeout := root.Duration("timeout")
	if timeout < 0 {
		timeout = 0
	}
	opts = append(opts, asf.WithHTTPClient(newHTTPClient(timeout)))
	return asf.NewClient(opts...)
}

func parseTimeFlag(cmd *cli.Command, name string) (time.Time, error) {
	value := strings.TrimSpace(cmd.String(name))
	if value == "" {
		return time.Time{}, nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse %s: %w", name, err)
	}
	return parsed, nil
}

func writeJSON(w io.Writer, products []asf.Product) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(products)
}

func printProductsTable(w io.Writer, products []asf.Product) {
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "SCENE\tPLATFORM\tSTART\tSTOP\tPATH\tURL")
	rows := 0
	for _, product := range products {
		props := product.Properties
		if isMetadataProduct(props) {
			continue
		}
		fmt.Fprintf(
			tw,
			"%s\t%s\t%s\t%s\t%d\t%s\n",
			props.SceneName,
			props.Platform,
			formatTime(props.StartTime),
			formatTime(props.StopTime),
			props.PathNumber,
			props.URL,
		)
		rows++
	}
	tw.Flush()
	if rows == 0 {
		fmt.Fprintln(w, "No downloadable products matched the filters. Try --output json for full results.")
	}
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.UTC().Format(time.RFC3339)
}

func convertSlice[T ~string](values []string) []T {
	var result []T
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			result = append(result, T(trimmed))
		}
	}
	return result
}

func isMetadataProduct(props asf.Properties) bool {
	if strings.EqualFold(props.ProcessingLevel, "METADATA") {
		return true
	}
	if strings.HasSuffix(strings.ToLower(props.URL), ".iso.xml") {
		return true
	}
	return false
}

func newHTTPClient(timeout time.Duration) *http.Client {
	jar, _ := cookiejar.New(nil)
	client := &http.Client{
		Timeout: timeout,
		Jar:     jar,
	}
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) == 0 {
			return nil
		}
		prev := via[len(via)-1]
		if auth := prev.Header.Get("Authorization"); auth != "" {
			req.Header.Set("Authorization", auth)
		}
		return nil
	}
	return client
}

func trimStrings(values []string) []string {
	var result []string
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
