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

	var configFile string = "conductor.yaml"
	flag.StringVar(&configFile, "config", configFile, "configurations file")
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

	// state

	lastViewChange conductrpc.ViewChange // last view change
	lastPhase      conductrpc.Phase      // last phase (can be restarted)

	// wait for
	phase      conductrpc.Phase    // wait for a phase
	viewChange config.ViewChange   // wait for a view change
	nodes      map[string]struct{} // wait starting nodes

	// remembered rounds: name -> round number
	rounds map[string]int64 // named rounds (the remember_round)
}

func (r *Runner) isWaiting() bool {
	return r.phase != 0 || !r.viewChange.IsZero() || len(r.nodes) > 0
}

func (r *Runner) acceptViewChange(vc conductrpc.ViewChange) (err error) {
	log.Print("view change:", vc.Round)
	// don't wait a VC
	if r.viewChange.IsZero() {
		r.lastViewChange = vc // just keep last
		return
	}
	// remember the round
	if r.viewChange.RememberRound != "" {
		r.rounds[r.viewChange.RememberRound] = vc.Round
	}
	if r.viewChange.ExpectMagicBlock.IsZero() {
		return // nothing more is here
	}
	if r.viewChange.ExpectMagicBlock.Round < vc.Round {
		return fmt.Errorf("expected view change %d")
	}
}

func (r *Runner) acceptPhase(phase conductrpc.Phase) {
	//
}

func (r *Runner) acceptAddMiner(mid conductrpc.MinerID) {
	//
}

func (r *Runner) acceptAddSharder(sid conductrpc.SharderID) {
	//
}

func (r *Runner) acceptMinerReady(mid conductrpc.MinerID) {
	//
}

func (r *Runner) acceptSharderReady(sid conductrpc.SharderID) {
	//
}

// Run the tests.
func (r *Runner) Run() (err error) {

	log.Println("start testing...")
	defer log.Println("end of testing")

	// for every test case
	for i, testCase := range r.conf.Tests {
		log.Printf("start %d %s test case", testCase.Name)
		for j, f := range testCase.Flow {
			log.Printf("  %d flow step", j)

			// proceed
			for r.isWaiting() {
				select {
				case vc := <-OnViewChange():
					err = r.acceptViewChange(vc)
				case p := <-OnPhase():
					err = r.acceptPhase(p)
				case mid := <-OnAddMiner():
					err = r.acceptAddMiner(mid)
				case sid := <-OnAddSharder():
					err = r.acceptAddSharder(sid)
				case mid := <-OnMinerReady():
					err = r.acceptMinerReady(mid)
				case sid := <-OnSharderReady():
					err = r.acceptSharderReady(sid)
				}
				if err != nil {
					return
				}
			}

			// execute
			if err = f.Execute(r); err != nil {
				return // fatality
			}
		}

		log.Printf("end of %d %s test case", testCase.Name)
	}

	return
}

//
// execute
//

// Start nodes, or start and lock them.
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
		r.nodes[n.ID] = struct{}{} // wait list
	}
	return
}

func (r *Runner) WaitViewChange(vc config.ViewChange) (err error) {
	r.viewChange = vc
}

func (r *Runner) WaitPhase(phase int, timeout time.Duration) (err error) {
	r.phase = phase
}

func (r *Runner) Unlock(names []string) (err error) {
	for _, name := range names {
		var n, ok = r.conf.Nodes.NodeByName(name) //
		if !ok {
			return fmt.Errorf("(start): unknown node: %q", name)
		}
		log.Print("node", n.Name, "unlock")
		r.server.UnlockNode(n.ID)
	}
}

func (r *Runner) Stop(names []string) (err error) {
	for _, name := range names {
		var n, ok = r.conf.Nodes.NodeByName(name) //
		if !ok {
			return fmt.Errorf("(start): unknown node: %q", name)
		}
		// TODO (sfxdx): send SIGINT to don't lock, or Stop asynchronously?
		log.Print("node", n.Name, "stopping...")
		if err = n.Stop(); err != nil {
			return
		}
		log.Print("node", n.Name, "stopped")
	}
}
