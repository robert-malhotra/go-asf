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
			&cli.StringFlag{Name: "platform", Usage: "satellite platform identifier"},
			&cli.StringFlag{Name: "beam-mode", Usage: "beam mode identifier"},
			&cli.StringFlag{Name: "polarization"},
			&cli.StringFlag{Name: "product-type"},
			&cli.StringFlag{Name: "processing-level"},
			&cli.StringFlag{Name: "flight-direction"},
			&cli.StringFlag{Name: "look-direction"},
			&cli.IntFlag{Name: "relative-orbit"},
			&cli.StringFlag{Name: "start", Usage: "start time (RFC3339)"},
			&cli.StringFlag{Name: "end", Usage: "end time (RFC3339)"},
			&cli.IntFlag{Name: "max-results", Value: 100},
			&cli.StringFlag{Name: "intersects", Usage: "WKT geometry"},
			&cli.StringSliceFlag{Name: "param", Usage: "additional key=value parameter"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			client, err := asf.NewClient()
			if err != nil {
				return err
			}

			builder := search.ParamsBuilder()
			params := builder
			if platform := cmd.String("platform"); platform != "" {
				params = params.Platform(search.Platform(platform))
			}
			if beam := cmd.String("beam-mode"); beam != "" {
				params = params.BeamMode(search.BeamMode(beam))
			}
			if pol := cmd.String("polarization"); pol != "" {
				params = params.Polarization(pol)
			}
			if pt := cmd.String("product-type"); pt != "" {
				params = params.ProductType(pt)
			}
			if level := cmd.String("processing-level"); level != "" {
				params = params.ProcessingLevel(level)
			}
			if dir := cmd.String("flight-direction"); dir != "" {
				params = params.FlightDirection(dir)
			}
			if look := cmd.String("look-direction"); look != "" {
				params = params.LookDirection(look)
			}
			if orbit := cmd.Int("relative-orbit"); orbit != 0 {
				params = params.RelativeOrbit(orbit)
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

			for iter.Next(ctx) {
				if err := enc.Encode(iter.Product()); err != nil {
					return err
				}
			}
			if err := iter.Err(); err != nil {
				return err
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
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			client, err := asf.NewClient()
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

			return client.DownloadProduct(ctx, product, cmd.String("dest"), opts...)
		},
	}
}

func lookupProduct(ctx context.Context, client *asf.Client, id string) (model.Product, error) {
	params := search.ParamsBuilder().Set("product_list", id).MaxResults(1).Build()
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
