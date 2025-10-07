package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/urfave/cli/v3"

	"github.com/example/go-asf/asf"
	"github.com/example/go-asf/asf/model"
	"github.com/example/go-asf/asf/search"
)

func main() {
	root := &cli.Command{
		Name:  "asfcli",
		Usage: "Search and download data from the ASF API",
		Commands: []*cli.Command{
			searchCommand(),
			downloadCommand(),
		},
	}

	if err := root.Run(context.Background(), os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func searchCommand() *cli.Command {
	return &cli.Command{
		Name:  "search",
		Usage: "Search for ASF products",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "platform", Usage: "satellite platform identifier (e.g. S1A or Sentinel-1A)"},
			&cli.StringFlag{Name: "beam-mode", Usage: "beam mode identifier"},
			&cli.StringFlag{Name: "polarization"},
			&cli.StringFlag{Name: "processing-level"},
			&cli.StringFlag{Name: "flight-direction"},
			&cli.StringFlag{Name: "look-direction"},
			&cli.IntFlag{Name: "relative-orbit"},
			&cli.StringFlag{Name: "dataset", Usage: "dataset short name"},
			&cli.StringSliceFlag{Name: "collection", Usage: "CMR collection concept ID"},
			&cli.StringSliceFlag{Name: "product", Usage: "product identifier to match"},
			&cli.StringSliceFlag{Name: "granule", Usage: "granule identifier to match"},
			&cli.StringFlag{Name: "start", Usage: "start time (RFC3339)"},
			&cli.StringFlag{Name: "end", Usage: "end time (RFC3339)"},
			&cli.IntFlag{Name: "max-results", Value: 100},
			&cli.StringFlag{Name: "intersects", Usage: "WKT geometry"},
			&cli.StringSliceFlag{Name: "param", Usage: "additional key=value parameter"},
			&cli.StringFlag{Name: "username", Usage: "Earthdata username for authenticated downloads", Sources: cli.EnvVars("ASF_USERNAME", "ASF_EARTHDATA_USERNAME")},
			&cli.StringFlag{Name: "password", Usage: "Earthdata password", Sources: cli.EnvVars("ASF_PASSWORD", "ASF_EARTHDATA_PASSWORD")},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			client, err := newClientFromCmd(cmd)
			if err != nil {
				return err
			}

			builder := search.ParamsBuilder()
			params := builder
			if platform := cmd.String("platform"); platform != "" {
				params = params.Platform(search.Platform(normalizePlatform(platform)))
			}
			if beam := cmd.String("beam-mode"); beam != "" {
				params = params.BeamMode(search.BeamMode(strings.ToUpper(beam)))
			}
			if pol := cmd.String("polarization"); pol != "" {
				params = params.Polarization(strings.ToUpper(pol))
			}
			if level := cmd.String("processing-level"); level != "" {
				params = params.ProcessingLevel(strings.ToUpper(level))
			}
			if dir := cmd.String("flight-direction"); dir != "" {
				params = params.FlightDirection(strings.ToUpper(dir))
			}
			if look := cmd.String("look-direction"); look != "" {
				params = params.LookDirection(strings.ToUpper(look))
			}
			if orbit := cmd.Int("relative-orbit"); orbit != 0 {
				params = params.RelativeOrbit(orbit)
			}
			if dataset := cmd.String("dataset"); dataset != "" {
				params = params.Dataset(dataset)
			}
			if collections := cmd.StringSlice("collection"); len(collections) > 0 {
				for _, c := range collections {
					if strings.TrimSpace(c) != "" {
						params = params.AddCollection(c)
					}
				}
			}
			if granules := cmd.StringSlice("granule"); len(granules) > 0 {
				params = params.GranuleList(granules...)
			}
			if products := cmd.StringSlice("product"); len(products) > 0 {
				params = params.ProductList(products...)
			}
			if start := cmd.String("start"); start != "" {
				t, err := time.Parse(time.RFC3339, start)
				if err != nil {
					return fmt.Errorf("parse start: %w", err)
				}
				params = params.StartTime(t)
			}
			if end := cmd.String("end"); end != "" {
				t, err := time.Parse(time.RFC3339, end)
				if err != nil {
					return fmt.Errorf("parse end: %w", err)
				}
				params = params.EndTime(t)
			}
			if max := cmd.Int("max-results"); max > 0 {
				params = params.MaxResults(max)
			}
			if geom := cmd.String("intersects"); geom != "" {
				params = params.IntersectsWith(geom)
			}

			built := params.Build()

			for _, kv := range cmd.StringSlice("param") {
				key, value, found := strings.Cut(kv, "=")
				if !found {
					return fmt.Errorf("invalid param %q, expected key=value", kv)
				}
				built.Add(key, value)
			}

			iter, err := client.Search(ctx, built)
			if err != nil {
				return err
			}

			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")

			wrote := false
			fmt.Print("[")
			first := true
			for iter.Next(ctx) {
				if !first {
					fmt.Print(",\n")
				}
				first = false
				if err := enc.Encode(iter.Product()); err != nil {
					return err
				}
				wrote = true
			}
			if err := iter.Err(); err != nil {
				return err
			}
			if wrote {
				fmt.Print("]\n")
			} else {
				fmt.Println("]")
			}
			return nil
		},
	}
}

func downloadCommand() *cli.Command {
	return &cli.Command{
		Name:  "download",
		Usage: "Download product files",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "product-id", Required: true},
			&cli.StringFlag{Name: "dest", Required: true},
			&cli.IntFlag{Name: "concurrency", Value: 2},
			&cli.BoolFlag{Name: "no-verify", Usage: "disable checksum verification"},
			&cli.StringFlag{Name: "username", Usage: "Earthdata username", Sources: cli.EnvVars("ASF_USERNAME", "ASF_EARTHDATA_USERNAME")},
			&cli.StringFlag{Name: "password", Usage: "Earthdata password", Sources: cli.EnvVars("ASF_PASSWORD", "ASF_EARTHDATA_PASSWORD")},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			client, err := newClientFromCmd(cmd)
			if err != nil {
				return err
			}

			product, err := lookupProduct(ctx, client, cmd.String("product-id"))
			if err != nil {
				return err
			}

			var opts []asf.DownloadOption
			if conc := cmd.Int("concurrency"); conc > 0 {
				opts = append(opts, asf.WithDownloadConcurrency(conc))
			}
			if cmd.Bool("no-verify") {
				opts = append(opts, asf.WithoutChecksum())
			}

			dest := cmd.String("dest")
			if err := os.MkdirAll(dest, 0o755); err != nil {
				return fmt.Errorf("create destination directory: %w", err)
			}

			return client.DownloadProduct(ctx, product, dest, opts...)
		},
	}
}

func lookupProduct(ctx context.Context, client *asf.Client, id string) (model.Product, error) {
	params := search.ParamsBuilder().ProductList(id).Build()
	iter, err := client.Search(ctx, params)
	if err != nil {
		return model.Product{}, err
	}
	if iter.Next(ctx) {
		return iter.Product(), nil
	}
	if err := iter.Err(); err != nil {
		return model.Product{}, err
	}
	return model.Product{}, errors.New("product not found")
}

func normalizePlatform(input string) string {
	trimmed := strings.TrimSpace(input)
	upper := strings.ToUpper(trimmed)
	switch upper {
	case "S1A":
		return string(search.PlatformSentinel1A)
	case "S1B":
		return string(search.PlatformSentinel1B)
	case "S1C":
		return string(search.PlatformSentinel1C)
	case "S1":
		return string(search.PlatformSentinel1)
	case "ALOS":
		return string(search.PlatformALOS)
	case "ERS1":
		return string(search.PlatformERS1)
	case "ERS2":
		return string(search.PlatformERS2)
	case "RADARSAT1", "RADARSAT-1", "RS1":
		return string(search.PlatformRADARSAT1)
	case "SMAP":
		return string(search.PlatformSMAP)
	case "UAVSAR":
		return string(search.PlatformUAVSAR)
	case "AIRSAR":
		return string(search.PlatformAIRSAR)
	case "SIRC":
		return string(search.PlatformSIRC)
	case "JERS", "JERS1":
		return string(search.PlatformJERS1)
	default:
		return trimmed
	}
}

func newClientFromCmd(cmd *cli.Command) (*asf.Client, error) {
	username := cmd.String("username")
	password := cmd.String("password")
	var opts []asf.Option
	if username != "" {
		opts = append(opts, asf.WithBasicAuth(username, password))
	}
	return asf.NewClient(opts...)
}
