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
			},
			&cli.IntFlag{
				Name:    numProvidersFlag,
				Usage:   "Number of needed registered providers",
				Aliases: []string{"np"},
			},
			&cli.IntFlag{
				Name:    numActiveSessionsFlag,
				Usage:   "Number of needed active sessions",
				Aliases: []string{"as"},
			},
			&cli.IntFlag{
				Name:    numInactiveSessionsFlag,
				Usage:   "Number of needed inactive providers",
				Aliases: []string{"is"},
			},
			&cli.IntFlag{
				Name:    numGoroutinesFlag,
				Usage:   "Number of goroutines",
				Value:   5000,
				Aliases: []string{"g"},
			},
			&cli.BoolFlag{
				Name:    cleanFlag,
				Usage:   "Clean directories before running",
				Aliases: []string{"c", "cl"},
			},
			&cli.BoolFlag{
				Name:    separateFlag,
				Usage:   "Separate progress bar each 1%",
				Aliases: []string{"sep", "s"},
			},
		},
		Action: func(cc *cli.Context) error {
			if err := setDefaultGenFlags(cc); err != nil {
				return err
			}
			if err := setupGenDirs(cc); err != nil {
				return err
			}

			sci, db, err := state.CreateStateContextAndDB(dirs.SciDir, dirs.SciLogDir, dirs.DbDir, nil)
			if err != nil {
				return err
			}

			var (
				sc      = magmasc.NewMagmaSmartContract()
				sFiller = filler.New(
					sci,
					sc,
					cc.Int(numGoroutinesFlag),
					cc.Bool(separateFlag),
				)
			)
			sc.SetDB(db)
			defer func() {
				if err := state.CloseSciAndDB(sci, db); err != nil {
					log.Println("Got error while closing databases ", err.Error())
				}
			}()

			return sFiller.Fill(
				cc.Int(numConsumersFlag),
				cc.Int(numProvidersFlag),
				cc.Int(numActiveSessionsFlag),
				cc.Int(numInactiveSessionsFlag),
			)
		},
	})
}

func setDefaultGenFlags(cc *cli.Context) error {
	if cc.Int(numConsumersFlag) == 0 && (cc.IsSet(numActiveSessionsFlag) || cc.IsSet(numInactiveSessionsFlag)) {
		if err := cc.Set(numConsumersFlag, "1"); err != nil {
			return err
		}
	}
	if cc.Int(numProvidersFlag) == 0 && (cc.IsSet(numActiveSessionsFlag) || cc.IsSet(numInactiveSessionsFlag)) {
		if err := cc.Set(numProvidersFlag, "1"); err != nil {
			return err
		}
	}
	return nil
}

func setupGenDirs(cc *cli.Context) error {
	if cc.Bool(cleanFlag) {
		if err := dirs.CleanDirs(); err != nil {
			return err
		}

	}
	return dirs.CreateDirs()
}
