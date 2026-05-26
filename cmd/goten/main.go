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
		Version: "0.1.0",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"c"},
				Value:   "goten.config.yaml",
				Usage:   "path to config file",
			},
			&cli.StringFlag{
				Name:  "env-file",
				Usage: "path to .env file (default: .env in current directory if it exists)",
			},
		},
		Commands: []*cli.Command{
			{
				Name:  "migrate",
				Usage: "Database migration commands",
				Commands: []*cli.Command{
					{
						Name:   "up",
						Usage:  "Apply all pending migrations",
						Action: cmdMigrateUp,
					},
					{
						Name:   "down",
						Usage:  "Roll back the last applied migration",
						Action: cmdMigrateDown,
					},
					{
						Name:   "status",
						Usage:  "Show applied and pending migrations",
						Action: cmdMigrateStatus,
					},
					{
						Name:      "generate",
						Usage:     "Generate a new migration template",
						ArgsUsage: "<name>",
						Action:    cmdMigrateGenerate,
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
