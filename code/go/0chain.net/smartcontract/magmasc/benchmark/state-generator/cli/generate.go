package cli

import (
	"log"

	"github.com/urfave/cli/v2"

	"0chain.net/smartcontract/magmasc"
	"0chain.net/smartcontract/magmasc/benchmark/dirs"
	"0chain.net/smartcontract/magmasc/benchmark/state-generator/filler"
	"0chain.net/smartcontract/magmasc/benchmark/state-generator/state"
)

func registerGenerateCommand(app *cli.App) {
	app.Commands = append(app.Commands, &cli.Command{
		Name:    "generate",
		Aliases: []string{"gen", "g"},
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:    numConsumersFlag,
				Usage:   "Number of needed registered consumers",
				Aliases: []string{"nc"},
				Value:   0,
			},
			&cli.IntFlag{
				Name:    numProvidersFlag,
				Usage:   "Number of needed registered providers",
				Aliases: []string{"np"},
				Value:   0,
			},
			&cli.IntFlag{
				Name:    numActiveSessionsFlag,
				Usage:   "Number of needed active sessions",
				Aliases: []string{"as"},
				Value:   0,
			},
			&cli.IntFlag{
				Name:    numInactiveSessionsFlag,
				Usage:   "Number of needed inactive providers",
				Aliases: []string{"is"},
				Value:   0,
			},
			&cli.IntFlag{
				Name:    goroutinesFlag,
				Usage:   "Number of goroutines",
				Aliases: []string{"g"},
				Value:   1000,
			},
			&cli.BoolFlag{
				Name:    cleanFlag,
				Usage:   "Clean directories before running",
				Aliases: []string{"c", "cl"},
			},
			&cli.BoolFlag{
				Name:    sepFlag,
				Usage:   "Separate progress bar each 1%",
				Aliases: []string{"sep", "s"},
			},
		},
		Action: func(cc *cli.Context) error {
			if err := setupGenDirs(cc); err != nil {
				return err
			}

			sci, db, err := state.CreateStateContextAndDB(dirs.SciDir, dirs.SciLogDir, dirs.DbDir, nil)
			if err != nil {
				return err
			}
			defer func() {
				if err = state.CloseSciAndDB(sci, db); err != nil {
					log.Println("Got error while closing databases ", err.Error())
				}
			}()

			sc := magmasc.NewMagmaSmartContract()
			sc.SetDB(db)

			return filler.New(sci, sc, cc.Int(goroutinesFlag), cc.Bool(sepFlag)).Fill(
				cc.Int(numConsumersFlag),
				cc.Int(numProvidersFlag),
				cc.Int(numActiveSessionsFlag),
				cc.Int(numInactiveSessionsFlag),
			)
		},
	})
}

func setupGenDirs(cc *cli.Context) error {
	if cc.Bool(cleanFlag) {
		if err := dirs.CleanDirs(); err != nil {
			return err
		}

	}
	return dirs.CreateDirs()
}
