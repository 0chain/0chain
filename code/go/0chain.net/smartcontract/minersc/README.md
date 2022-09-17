Miner SC
========

# Unit tests

## Test in docker container

From 0chain project root execute the following command to run unit-tests

```
./docker.local/bin/sc_unit_test.sh 0chain.net/smartcontract/minersc
```


# Node settings

#### Example

```yaml
delegate_wallet: '<put_wallet_id_here>'
service_charge: 0.10
number_of_delegates: 10
min_stake: 10.0
max_stake: 1000.0
```

#### Delegate wallet

Delegate wallet is wallet_id of user that can control node changing its
settings. If the delegate_wallet is empty, then node ID used. For example, a
genesis node can't have a delegate wallet, because to register a wallet the
genesis nodes should be running (e.g. already registered). Thus, the
delegate_wallet can be set for non-genesis nodes only. For genesis nodes
their IDs used. Check out b0mnode1_keys.txt for example for miner 1. Also,
a node prints its ID in logs on start. For miner 1 it's wallet file is
```
{
  "client_id": "31810bd1258ae95955fb40c7ef72498a556d3587121376d9059119d280f34929",
  "client_key": "255452b9f49ebb8c8b8fcec9f0bd8a4284e540be1286bd562578e7e59765e41a7aada04c9e2ad3e28f79aacb0f1be66715535a87983843fea81f23d8011e728b",
  "keys": [
    {
      "public_key": "255452b9f49ebb8c8b8fcec9f0bd8a4284e540be1286bd562578e7e59765e41a7aada04c9e2ad3e28f79aacb0f1be66715535a87983843fea81f23d8011e728b",
      "private_key": "8a3f56841f7fd5feb058cae8d16ab87e1c682fd53cfc1204a80c5d0ceb15f509"
    }
  ],
  "version": "1.0",
  "date_created": "2020-03-16 00:47:58.247961953 +0400 +04 m=+0.015793530"
}
```
The delegate wallet can't be changed even if node ID used instead.

The `mn-update-config` command of the _zwallet_ should be called by owner of the
delegate wallet. Otherwise, command will fail.

#### Service charge.

Service charge is % (value in [0; 1) range) of all fees and rewards of a block
that goes to block generator stake holders.

The formulas
```
generator_fees = all_fees * service_charge
generator_rewards = all_rewards * service_charge
```

Thus, generator receives `generator_fees`, `generator_rewards` and plus
share_ratio of rest (see _share_ratio_ below).

#### Number of delegates.

Number of delegates is max number of stake pools can be created for the node.
Positive integer.

#### Min stake, Max stake.

The min stake and the max stake are stake boundaries. This min/max values
should not conflict with min/max_stake configured for entire SC (in sc.yaml).
Otherwise the node can't be registered. Measured in tokens (101.12 for example).

# Miner SC settings

```yaml
  minersc:
    max_n: 100
    min_n: 3
    sharders_max_n: 0.30 # 30%
    sharders_min_n: 0.30 # 30%
    t_percent: .51
    k_percent: .75
    min_stake: 0.01
    max_stake: 100.0
    start_rounds: 50
    contribute_rounds: 50
    share_rounds: 50
    publish_rounds: 50
    wait_rounds: 50
    interest_rate: 0.001
    reward_rate: 1.0
    share_ratio: 0.10
    block_reward: 0.7
    max_charge: 0.5
    epoch: 15000000
    reward_decline_rate: 0.1
    interest_decline_rate: 0.1
    max_mint: 4000000.0
```

#### Min stake, Max stake.

Min and max stake boundaries set for entire SC. A node can't set their
min/max_stake out of this SC boundary.

#### Interest rate.

Interest rate is % (value in [0; 1) range) that used to calculate interests of
stake holders. The interest is payed periodically, on rounds that are a multiple of
the `sc.yaml minersc.reward_round_frequency` settings.

The formula
```
interest_rewrds (mint) = stake_capacity * interest_rate
```

The interest_rate decreased by `interest_decline_rate` every epoch (see _epoch_
below).


#### Reward rate.

Reward rate is initially 1.0. It's % (value in [0; 1) range) that used to
decline block rewards every epoch. Initially it's 1.0 (100%). E.g. a generator
gives 100% of rewards (see _block_reward_ below).

The formula
```
block_reward (mint) = reward_rate * block_reward
```

Every _epoch_ the reward_rate declined by reward_decline_rate. And the reward
declined with it.

#### Share ratio.

After a generator subtracts its service_charge the rest of fees divided by
block sharders (stake holders) and the generator stake holders by the
share_ratio.

The formula
```
generator_rewards = block_reward * share_ratio
sharders_rewards  = block_reward - generator_rewards

generator_reward_service_charge = generator_rewards * service_charge
sharder_reward_service_charge = sharders_rewards * service_charge

generator_fees = block_fees * share_ratio
sharders_fees  = block_fees - generator_fees

generator_fee_service_charge = generator_fees * service_charge
sharder_fee_service_charge = sharders_fees * service_charge
```

The _generator_rewards_ and _generator_fees_ divided between generator's stake
holders depending their stake capacities.

The sharders_rewards and sharders_fees divided between all block sharders
equally and this equal parts divided between sharders' stake holder depending
their stake capacities.

#### Block reward.

Even if fee is zero, a generator receive block_reward (minted). The block_reward
measured in tokens (0.7 tokens, for example). The real block_reward depends on
reward_rate.

The formula
```
block_reward (mint) = block_reward (configured) * reward_rate
```

Since, the reward_rate is declining every epoch, the block_reward is declining
too.

#### Max charge

Max charge is max possible service_charge can be set by a node. E.g. if a node
provide service_charge greater then this, than it can't be registered.

#### Epoch, Reward decline rate, Interest decline rate.

The Epoch is number of round to decline _reward_rate_ and _interest_rate_. The
_reward_decline_rate_ and _interest_decline_rate_ used.

The formula
```
new_reward_rate = current_reward_rate * (1.0 - reward_decline_rate)
new_interest_rate = current_interest_rae * (1.0 - interest_decline_rate)
```

The _reward_decline_rate_ and _interest_decline_rate_ measured in values
in [0; 1) range. For example, if _interest_decline_rate_ is 0.1, then every
epoch _interest_rate_ becomes 90% of its value. For example, _interest_rate_
= 0.1, with _interest_decline_rate_ = 0.1, after the first epoch declining
becomes 0.09. Next epoch it becomes 0.081. Next one 0.0729, etc.

Set a decline_rate to 0.5 to got the "half" every epoch.

#### Max mint

The max_mint used to stop any minting by entire SC after all SC mints reaches
the max_mint. It's measured in tokens. The mints are
- the block reward
- stake interests

There is `minetd` field in the _mn-config_ zwallet command that shows amount
of tokens minted by Miner SC for current time.

# Stake pools lifecycle.

When a stake pool created it becomes PENDING. Next View Change it becomes
ACTIVE and starts collecting interests. When user deletes his stake the
stake pool becomes DELETING and waits next View Change. The next View Change
the pool will be unlocked.

A PENDING pool can be unlocked immediately.

If a node leaves blockchain (leaves Magic Block) then Miner SC unlocks all
stakes of the node returning tokens to owners.

All interests and rewards payed directly to stake holders' wallets.

It's impossible to make a stake for a offline node (any node doesn't
participate blockchain, e.g. any node not from current magic block). Since,
this nodes treated as offline and their tokens unlocked. Thus, (1) a node can't
join blockchain, (2) still not join (a just started node) (3) leaves blockchain
(turned off node) all are treated as offline. The Miner SC unlocks all stakes
of all offline nodes returning tokens back.

# Related zwallet commands

```
  mn-config          Get miner SC global info.
  mn-user-info       Get list of user pools.
  mn-info            Get miner/sharder info from Miner SC.
  mn-lock            Add miner/sharder stake.
  mn-pool-info       Get miner/sharder pool info from Miner SC.
  mn-unlock          Unlock miner/sharder stake.
  mn-update-settings Change miner/sharder settings in Miner SC.
```

# Step by step guide.

1. If 0chain has updated, cleanup blockchain and rebuild all.
2. Start sharder 1 and 1-3 miners (genesis nodes).
3. Create wallet in zwallet and fill if with tokens
    ```
    for run in {1..10}; do ./zwallet faucet --methodName pour --input “{Pay day}”; done
    ```
   This takes a while.
4. Determine ID of the wallet
    ```
    cat ~/.zcn/wallet.json
    ```
   The "client_id" field is the wallet ID.
5. Check out Miner SC configurations to determine min_stake and max_sake of SC.
6. Configure 5th miner (file `docker.local/config/0chain.yaml`)
    ```
    delegate_wallet: '<put the wallet ID here>'
    service_charge: 0.10
    number_of_delegates: 10
    min_stake: 0.1
    max_stake: 10.0
    ```
7. Start the 5th miner and wait some time to let the miner register in Miner SC.
   Export the miner 5 ID to use in the commands
    ```
    export MINER5=53add50ff9501014df2cbd698c673f85e5785281cebba8772a64a6e74057d328
    ```
   Wait view change to let the 5th miner join blockchain.
8. Stake 10 tokens for the 5th miner
    ```
    ./zwallet mn-lock --id $MINER5 --tokens 10.0
    ```
   end export returned pool id
    ```
    export POOL=<the returned pool ID>
    ```
9. Check out user (own) pools
    ```
    ./zwallet mn-user-info
    ```
10. Check 5th miner information. It should contain the pool.
    ```
    ./zwallet mn-info --id $MINER5
    ```
11. Check out the pool.
    ```
    ./zwallet mn-pool-info --id $MINER5 --pool_id $POOL
    ```
    Make sure status is ACTIVE, or wait a view change and check again.
    A pool becomes ACTIVE after next view change its created. After it becomes
    ACTIVE check own balance that should receive interests for the sake.
12. Check out your balance. The total_paid" of above command and the balance
    should be closer.
13. It's possible to make a stake for genesis nodes too.
14. Delete the stake
    ```
    ./zwallet mn-unlock --id $MINER5 --pool_id $POOL
    ```
    The stake will be unlocked next view change. Check out the pool
    ```
    ./zwallet mn-pool-info --id $MINER5 --pool_id $POOL
    ```
    It's status should be "DELETING".
15. Wait a view change to let the stake be unlocked. Check the pool (should
    be not found)
    ```
    ./zwallet mn-pool-info --id $MINER5 --pool_id $POOL
    ```
    Check the node, total_stake should be zero.
    ```
    ./zwallet mn-info --id $MINER5
    ```
    Check user pools, the list should b empty
    ```
    ./zwallet mn-user-info
    ```
16. Lock stake again.
    ```
    ./zwallet mn-lock --id $MINER5 --tokens 10.0
    ```
    Wait the pool becomes ACTIVE. Turn off the 5th miner. Wait View Change
    again. The pool should be unlocked and all tokens returned to user.

# Client specific API

Use `./zwallet mn-user-info` to get all stake pools of current user.
