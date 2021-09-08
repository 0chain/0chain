package cli

import (
	"log"

	"github.com/urfave/cli/v2"

	"0chain.net/smartcontract/magmasc"
	"0chain.net/smartcontract/magmasc/benchmark/dirs"
	"0chain.net/smartcontract/magmasc/benchmark/sessions"
	"0chain.net/smartcontract/magmasc/benchmark/state-generator/state"
)

func registerStatusCommand(app *cli.App) {
	app.Commands = append(app.Commands, &cli.Command{
		Name:    "status",
		Aliases: []string{"stat", "s"},
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    numActiveSessionsFlag,
				Usage:   "Active sessions counter",
				Aliases: []string{"nas", "a"},
			},
			&cli.BoolFlag{
				Name:    numInactiveSessionsFlag,
				Usage:   "Inactive sessions counter",
				Aliases: []string{"nis", "i"},
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

			if cc.Bool(numActiveSessionsFlag) {
				nas, err := sessions.CountActive(sc, sci)
				if err != nil {
					return err
				}
				log.Printf("Active sessions: %v", nas)
			}
			if cc.Bool(numInactiveSessionsFlag) {
				nis, err := sessions.CountInactive(sc, sci)
				if err != nil {
					return err
				}
				log.Printf("Inactive sessions: %v", nis)
			}

			return nil
		},
	})
}
