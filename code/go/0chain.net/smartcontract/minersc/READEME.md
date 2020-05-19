Miner SC
========

# Node settings

```yaml
# delegate wallet is wallet that used for all rewards of a node (miner/sharder);
# if delegate wallet is not set, then node id used; a delegate wallet can't
# be changed
delegate_wallet: ''       # delegate wallet for all rewards

# % of fees for generator
service_charge: 0.10      # [0; 1) of all fees

# max number of delegate pools allowed by a node in miner SC
number_of_delegates: 10   # max number of delegate pools

# min stake pool amount allowed by node; should not conflict with
# SC min_stake 
min_stake: 10.0    # tokens

# max stake pool amount allowed by node; should not conflict with
# SC max_stake 
max_stake: 1000.0  # tokens
```

# Miner SC settings

```yaml
  minersc:
    max_n: 100
    min_n: 3
    t_percent: .51
    k_percent: .75

    # min stake can be set by a node (boundary for all nodes)
    # a node can't provide min_stake less then this one
    min_stake: 0.01 

	# max stake can be set by a node (boundary for all nodes)
	# a node can't provide max_stake greater then this one
    max_stake: 100.0

    start_rounds: 50
    contribute_rounds: 50
    share_rounds: 50
    publish_rounds: 50
    wait_rounds: 50

    # stake interests, will be declined every epoch
    interest_rate: 0.001 # [0; 1)

    # reward rate for generators, will be declined every epoch
    reward_rate: 1.0 # [0; 1)

    # share ratio is miner/block sharders rewards ratio, for example 0.1
    # gives 10% for miner and rest for block sharders; the rewards calculated
    # after a generator subtracts its service_change
    share_ratio: 0.10 # [0; 1)

    # reward for a block, a generator got this reward (mint) for a block;
    # this reward will be declined by reward_decline_rate and by the
    # reward_rate (initially 1 - 100%)
    block_reward: 0.7 # tokens

    # max service charge can be set by a generator, if a miner provides
    # service charge greater then this one, then it can't be registered
    max_charge: 0.5 # %

    # epoch is number of rounds before rewards and interest are decreased
    epoch: 15000000 # rounds

    # decline rewards every new epoch by this value (the block_reward)
    reward_decline_rate: 0.1 # [0; 1), 0.1 = 10%

    # decline interests every new epoch by this value (the interest_rate)
    interest_decline_rate: 0.1 # [0; 1), 0.1 = 10%

    # no mints after this boundary (for entire SC); real max_mint can be
    # a bit greater depending on last mint amount
    max_mint: 4000000.0 # tokens
```

# Related zwallet commands

```
  mn-config          Get miner SC global info.
  mn-info            Get miner/sharder info from Miner SC.
  mn-lock            Add miner/sharder stake.
  mn-pool-info       Get miner/sharder pool info from Miner SC.
  mn-unlock          Unlock miner/sharder stake.
  mn-update-settings Change miner/sharder settings in Miner SC.
```

# Rewards

### Epoch

Every epoch reward_rate and interest_rate declined by reward_decline_rate and
interest_decline_rate configurations.

### Max mint.

While the total amount of tokens minted by miner SC reaches the max_mint, then
miner SC no mints anymore.

### Block rewards.

For a block its generator receive:

- `reward (mint) * reward_rate`
- `service_charge * fees`

Rest of fees divided between generator's stake holders and block sharders by
'share_ratio'

```
miner_stake_holders   = rest_of_fees * share_ratio
block_sharders_reward = rest_of_fees - miner_stake_holders
```

For the stake holder their reward divided depending stake amount.
For sharders block_sharders_reward divided equally.

### Stake

- A stake starts 'work' next VC after adding. For every VC node is online
  the stake receive interests. Every VC.
    ```
    stake_interests = stake * interests_rate.
    ```

- If node goes offline then stake (all stakes of the node) will be unlocked.

- If user want to unlock a stake (using mn-unlock) then the stake will be unlocked
after VC after the unlock request.
