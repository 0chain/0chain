// Package conductor represents 0chain BC testing conductor that
// maintain BC joining and leaving. It's introduced to test some
// view change cases where a miner comes up and goes down.
//
// The conductor uses RPC to control nodes. It starts and stops
// miners and sharders. It controls their lifecycle and entire
// system state. There is internal BC monitoring to generate
// events depending BC state (view change, view change phase,
// nodes registration, etc).
//
// All the cases uses b0magicBlock_4_miners_1_sharder.json where
// there is 1 genesis sharder and 4 genesis miners. Also, there is
// 2 non-genesis sharders and 1 non-genesis miner.
package main

import (
	"flag"
	"log"
	"os"
	"time"

	"gopkg.in/yaml.v2"

	"0chain.net/conductor/conductrpc"
	"0chain.net/conductor/config"

	"github.com/kr/pretty"
)

func main() {

	var (
		configFile string = "conductor.yaml"
		verbose    bool   = false
	)

	flag.StringVar(&configFile, "config", configFile, "configurations file")
	flag.BoolVar(&verbose, "verbose", verbose, "verbose logs")
	flag.Parse()

	var conf = readConfig(configFile)

	//

	_ = conf

	pretty.Print(conf)

	var _ conductrpc.Server
}

func readConfig(configFile string) (conf *config.Config) {
	conf = new(config.Config)
	var fl, err = os.Open(configFile)
	if err != nil {
		log.Fatalf("opening configurations file %s: %v", configFile, err)
	}
	defer fl.Close()
	if err = yaml.NewDecoder(fl).Decode(conf); err != nil {
		log.Fatalf("decoding configurations file %s: %v", configFile, err)
	}
	return
}

type Runner struct {
	server *conductrpc.Server
	conf   *config.Config

	verbose bool             // verbose logs
	rounds  map[string]int64 // named rounds (the remember_round)
}

func (r *Runner) Run() (err error) {
	for i, test := range r.conf.Tests {
		if err = r.runTest(i, test); err != nil {
			return
		}
	}
	return
}

func (r *Runner) runTest(i int, test config.Case) (err error) {
	log.Print("start %d %s", i, test.Name)
	defer log.Print("end %d %s", i, test.Name)

	for _, f := range test.Flow {
		f.Execute(ex)
	}

}

func (r *Runner) Start(names []string, lock bool) (err error) {
	// start nodes
	for _, name := range names {
		var n, ok = r.conf.Nodes.NodeByName(name) //
		if !ok {
			return fmt.Errorf("(start): unknown node: %q", name)
		}
		r.server.AddNode(n.ID, lock)
		if err = n.Start(r.conf.Logs); err != nil {
			return
		}
	}
	// wait nodes
	var nm = make(map[string]struct{}, len(names))
	for _, n := range names {
		nm[n] = struct{}{}
	}

	// TODO (sfxdx): add timeout for nodes to start
	for len(nm) > 0 {
		select {
		case minerID := <-r.server.OnAddMiner():
			if _, ok := nm[string(minerID)]; !ok {
				return fmt.Errorf("unexpected miner added: %q", minerID)
			}
			delete(mn, string(minerID))
			log.Print("%q started", minerID)
		case sharderID := <-r.server.OnAddSharder():
			if _, ok := nm[string(sharderID)]; !ok {
				return fmt.Errorf("unexpected sharder added: %q", sharderID)
			}
			delete(mn, string(sharderID))
			log.Print("%q started", sharderID)
		}
	}

	return
}

func (r *Runner) WaitViewChange(vc config.ViewChange) (err error) {
	//
}

func (r *Runner) WaitPhase(phase int, timeout time.Duration) (err error) {
	//
}

func (r *Runner) Unlock(names []string) (err error) {
	//
}

func (r *Runner) Stop(names []string) (err error) {
	//
}
