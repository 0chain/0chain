Miner SC
========

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
stake holders. The interests payed every view change.

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
generator_reward_service_charge = block_reward * service_charge
generator_fee_service_charge    = block_fees * service_charge

rest_reward = block_reward - generator_reward_service_charge
rest_fees = block_fees - generator_fee_service_charge

generator_rewards = rest_reward * share_ratio
sharders_rewards  = rest_reward - generator_rewards

generator_fees = rest_fees * share_ratio
sharders_fees  = rest_fees - generator_fees
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
