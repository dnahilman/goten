package main

import (
	"context"
	"fmt"
	"os"

	"github.com/urfave/cli/v3"
)

func main() {
	cmd := &cli.Command{
		Name:    "goten",
		Usage:   "Goten authentication CLI",
		Version: "0.2.0",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"c"},
				Value:   "goten.config.yaml",
				Usage:   "path to config file",
			},
		},
		Commands: []*cli.Command{
			{
				Name:   "generate",
				Usage:  "Generate ORM models from the active plugins' schema (run db.AutoMigrate on them)",
				Action: cmdGenerate,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "output",
						Aliases: []string{"o"},
						Usage:   "output file (default: <generate.output_dir>/auth_models.go)",
					},
					&cli.StringFlag{
						Name:    "package",
						Aliases: []string{"p"},
						Usage:   "override the generated package name",
					},
					&cli.BoolFlag{
						Name:    "yes",
						Aliases: []string{"y"},
						Usage:   "overwrite an existing file without prompting",
					},
				},
			},
		},
	}
	if err := cmd.Run(context.Background(), os.Args); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
