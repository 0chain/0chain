package config

// The Flow represents single value map.
//
//     start            - list of 'sharder 1', 'miner 1', etc
//     wait_view_change - remember_ronnd and/or expect_magic_block
//     start_lock       - see start
//     wait             - wait for a phase
//     unlock           - see start
//     stop             - see start
//
// See below for a possible map formats.
type Flow map[string]interface{}

// Flows represents order of start/stop miners/sharder and other BC events.
type Flows []Flow

// A Case represents a test case.
type Case struct {
	Name string `json:"name" yaml:"name" mapstructure:"name"`
	Flow Flows  `json:"flow" yaml:"flow" mapstructure:"flow"`
}

type Config struct {
	// Address is RPC server address
	Address string `json:"address" yaml:"address" mapstructure:"address"`
	// Cases is test cases.
	Cases []Flow `json:"cases" yaml:"cases" mapstructure:"cases"`
}

/*

Example configurations
======================

address: 127.0.0.1:15210

working_directory: ".."

nodes:
  - name: "sharder 1"
  	id: 57b416fcda1cf82b8a7e1fc3a47c68a94e617be873b5383ea2606bda757d3ce4
    work_dir: "docker.local/sharder1"
    start_command: "../bin/start.b0sharder.sh"

  - name: "sharder 2"
  	id: b098d2d56b087ee910f3ee2d2df173630566babb69f0be0e2e9a0c98d63f0b0b
    work_dir: "docker.local/sharder2"
    start_command: "../bin/start.b0sharder.sh"

  - name: "sharder 3"
  	id: d9558143f8e976126367603bff34125f5eb94720df8d7acefffdd66675d134c2
    work_dir: "docker.local/sharder3"
    start_command: "../bin/start.b0sharder.sh"

  - name: "miner 1"
  	id: 31810bd1258ae95955fb40c7ef72498a556d3587121376d9059119d280f34929
    work_dir: "docker.local/miner1"
    start_command: "../bin/start.b0miner.sh"

  - name: "miner 2"
  	id: 585732eb076d07455fbebcf3388856b6fd00449a25c47c0f72d961c7c4e7e7c2
    work_dir: "docker.local/miner2"
    start_command: "../bin/start.b0miner.sh"

  - name: "miner 3"
  	id: bfa64c67f49bceec8be618b1b6f558bdbaf9c100fd95d55601fa2190a4e548d8
    work_dir: "docker.local/miner3"
    start_command: "../bin/start.b0miner.sh"

  - name: "miner 4"
  	id: 8877e3da19b4cb51e59b4646ec7c0cf4849bc7b860257d69ddbf753b9a981e1b
    work_dir: "docker.local/miner4"
    start_command: "../bin/start.b0miner.sh"

  - name: "miner 5"
  	id: 53add50ff9501014df2cbd698c673f85e5785281cebba8772a64a6e74057d328
    work_dir: "docker.local/miner5"
    start_command: "../bin/start.b0miner.sh"

flows:
  - name: "prepare"
    flow:
    - start:
      - sharder 1
    - start:
      - miner 1
      - miner 2
      - miner 3

  - name: "miner 4 comes up on phase 0"
    flow:
    - wait_view_change:
        timeout: 10m
        remember_round: "starting_round"
    - start_lock:
      - miner 4
    - wait:
      phase: 0
      timeout: 1 minute
    - unlock:
      - miner 4
    - wait_view_change:
        timeout: 10 minute
        expect_magic_block:
          round_next_vc_after: "starting_round"
          sharders:
            - sharder 1
          miners:
            - miner 1
            - miner 2
            - miner 3
            - miner 4

  - name: "miner 4 goes down until next view change"
    flow:
    - wait_view_change:
        timeout: 10m
        remember_round: "starting_round"
    - wait:
      phase: 0
      timeout: 1 minute
    - stop:
      - miner 4
    - wait_view_change:
        timeout: 10 minute
        expect_magic_block:
          round_next_vc_after: "starting_round"
          sharders:
            - sharder 1
          miners:
            - miner 1
            - miner 2
            - miner 3

*/
