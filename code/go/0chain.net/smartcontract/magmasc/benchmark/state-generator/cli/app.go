package cli

import (
	"context"
	"log"
	"os"

	"github.com/urfave/cli/v2"
)

// New returns constructed cli application instance.
func New() *cli.App {
	// construct cli application
	app := &cli.App{
		Usage: "App for generating databases for benchmark testing",
		ExitErrHandler: func(_ *cli.Context, err error) {
			if err != nil {
				log.Fatalln(err)
			}
		},
	}

	// register commands into cli application
	registerGenerateCommand(app)
	registerStatusCommand(app)

	return app
}

// Start starts the application.
func Start(ctx context.Context, app *cli.App) error {
	return app.RunContext(ctx, os.Args)
}
