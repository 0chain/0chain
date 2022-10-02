STORAGE SC
==========

# Unit tests

## Test in docker container.

From 0chain project root execute the following command to run unit-tests

```
./docker.local/bin/sc_unit_test.sh 0chain.net/smartcontract/storagesc
```

# Time unit.

The Storage SC has time_unit configuration. Once applied it can't be changed
then (excluding blockchain full reset case, that works). All write prices is
measured in tokens / GB / time_unit. E.g. tokens for size for some duration.

If user creates allocation for, say, 4 days. And uploads a file. Then the file
is uploaded for 4 days (rest of the allocation). If user deletes the file,
then part of tokens moved back to the user (to one of his write pools).

If time_unit configured as 48h, then for a write_price 1 tok / GB / time_unit
the user pays 2 tok (4 days = 2 * 48h). That's all.

The time_unit configured in sc.yaml in storagesc part. It can be given by REST
API as other SC configurations.

# Flow

## Blobber

### 1. Blobber: send transaction -> registration

Provide

 - capacity                   bytes
 - min_lock_demand            [0; 1]
 - write_price                tok / GB
 - read_price                 tok / GB (by 64 KB chunks)
 - max_offer_duration         time.Duration
 - challenge_completion_time  time.Duration
 - delegate_wallet            string

The transaction also updates blobber's 'last health check' making it healthy.

### 2. SC: check capacity, the registration transaction handling

##### Zero capacity

There is special case where capacity is 0 -> remove blobber from SC.
For case, where blobber doesn't provide its service anymore. Thus the
blobber will not be selected by SC for users' requests. But the blobber
still should provide its service for all its opened offers (allocations).
This allocations can't be extended by size or expiration time but can be
reduced.

##### Non zero capacity

Validate blobber terms.

SC has configured max value for blobebrs' min_lock_demand. Min value for
max_offer_duration. Max value for challenge_completion_time.

###### Blobber already registered. Update blobber

Check required capacity stake, lock more tokens if missing. Or release
some tokens if overfill.

```
required_stake = (capacity / GB) * write_price
difference = stake_pool_tokens - required_stake
```

###### Blobber not registered or has removed

Create related stake pool. Lock required tokens.

```
required_stake = (capacity / GB) * write_price
```

## Pools

## User

### 1. Create wallet

User creates wallet by zwalelt or zbox.

### 2. Create read pool

Zwallet or zbox creates read pool creating wallet. If the wallet created a
long time ago, or read pool creating fails for a reason, then user can create
read pool manually. Otherwise, the user can't use storage SC to create an
allocation.

## Create allocation

### 1. Send request transaction

Provide

- read_price range
- write_price range
- allocation size
- allocation expiration time
- preferred blobbers, optional
- data shards
- parity shards

- number or tokens to lock

### 2. SC select blobbers for the ranges

- get all (or preferred) blobbers, filters them by provided values
  (regardless number of tokens) and selects random of them
- determine required number of tokens
- create related challenge pool (not implemented yet)
- determine overall challenge_completion_time that is max
  challenge_completion_time of all blobbers selected
- determine overall min lock demand
- create related write pool (expiration + the challenge_completion_time)
- fill the write pool with all provided tokens; if user provides less tokens
  then transaction fails; if user provides more tokens, then all the tokens
  locked in the write pool
- add info about this allocation to blobbers' stake pools
- add info about blobbers' size used

So, on success, we have

- allocation_id
- list of blobbers (specific random order)
   - blobber id
   - size used by this allocations
   - terms of the blobber
- user_id
- allocation size
- challenge completion time (maximum, for the write and challenge pools)
- expiration
- overall min lock demand

## Update allocation

Only size or expiration of allocation can be changed. If one of values is
increased, then it's about extending, even if one another is reduced.

If size or expiration reduced, then it's about reducing. In this case we
are using the same terms. In extending, we are using new terms regarding
blobbers.

In this request user provides difference for expiration and difference for size.
E.g. +/- some time, and +/- some size.

### Extend allocation

1. Get related blobbers
2. Get new terms of them, update in allocation
3. Calculate new min lock demand
4. Calculate new challenge_completion_time
5. Lock more user's tokens if required
6. Update expiration of write pool
7. Update information about capacity used in blobbers
8. Update information about capacity used in blobbers' stake pools

If a blobber doesn't have enough capacity, then it fails. Same, if user's
write pool (+ this extending request) doesn't have enough tokens to extend.

Also, allocation can't be extended wide then max_offer_duration of a blobber
from time of this extending transaction.

Terms used for extended allocation calculated using existing allocation
terms and new terms of blobbers (if related blobbers changes their terms)
using weighted average. Where weight is `size*period`. For example

1. create allocation for 2 days
2. after 1 day, extend it for 3 days

```
|    1 day    |    2 day    |    3 day    |
[<----create allocation---->]
             [<-----extend allocation---->]
```
Thus 1st weight is (2days * size) and second weight is (1day * size). E.g.
weight for extension is for the 3rd day only.

### Reduce allocation. Close allocation.

The same terms used.

1. Get related blobbers
2. Update expiration of write pool if expiration changed
3. Update information about capacity used in blobbers, if released
4. Update information about capacity used in blobbers' stake pools, if released

If user reduces expiration time making in expired, then it closes allocation
and allocation will be active challenge_completion_time and closed then.

User can't extend size and close allocation. Such transactions are invalid.

A size reducing doesn't reduce min_lock_demand to prevent the salvation attack.

### Cancel allocation.

If blobbers doesn't work in reality, then an allocation can't be used.
Allocation owner can perform cancel_allocation transaction to close the
allocation and return back all funds. In this case, blobbers doesn't
receive their min_lock_demand.

### Finalize allocation.

When allocation expired, it should be finalized. Blobbers runs the finalization
automatically. And user doesn't need to do anything. But, if blobbers doesn't
do it, then user (allocation owner) can perform finalize_allocation transaction.
The transaction:

- makes sure all blobbers got their min_lock_demand (excluding penalty)
- unlocks all tokens in write pool, moving them back to user
- moves all tokens of a challenge pool to user, if any
- marks allocation a finalized


# Setup

## Order

 1. clean blockchain
 2. clean blobbers
 3. rebuild all applications: zboxcli, zwallet, blobbers, sharder, miners
 4. start sharder
 5. start miners
 6. remove wallet.json and allocation.txt
 7. configure blobbers (remember capacity, write_price, id)
 8. create new wallet (zwallet), it creates read pool automatically
 9. add some tokens to the wallet (`./zwallet faucet --methodName pour --input “{Pay day}”`)
10. find out your wallet ID
    ```
    cat ~/.zcn/wallet.json
    ```
    The client_id is the wallet ID.
11. configure blobbers and validators (since they use the same stake pool
    for rewards) making your wallet their delegate_wallet. E.g. edit
    ```
     config/0chain_blobber.yaml
    ```
    and
    ```
    config/0chain_validator.yaml
    ```
    in blobbers repository and add/set delegate_wallet setting to you wallet ID.
12. start the blobbers and validators
13. make a stake for the blobbers to allow them accept allocations
    ```
    ./zbox sp-lock --blobber_id f65af5d64000c7cd2883f4910eb69086f9d6e6635c744e62afcfab58b938ee25 --tokens 10
    ./zbox sp-lock --blobber_id 7a90e6790bcd3d78422d7a230390edc102870fe58c15472073922024985b1c7d --tokens 10
    ```
14. create new allocation providing tokens to write pool
15. check out write_pool
16. check out blobbers stake pools
17. etc, see below for example

## Step by step

Note: some commands implemented in zbox , since they belongs to storage SC. Feel
free to use these zbox command.

### Initial

1. Start Sharder. Wait it. Start miners. Wait blockchain starts.

2. Create wallet, read pool, get some tokens
    ```
    for run in {1..20}
    do
        ./zwallet faucet --methodName pour --input “{Pay day}”
    done
    ```
    This command does it all.

3. Configure blobbers and validators setting them own wallet as delegate_wallet.
   Use cat ~/.zcn/wallet.json and use client_id field that is the wallet ID
   and set it to 'delegate_wallet' setting of blobbers and validators.
   Since, blobbers and validators uses the same delegate_wallets for rewards
   the first registered will be used. E.g. it's important to set the
   delegate_wallet for validators, because they registers fist as rule.
   Example blobber configurations
    ```yaml
    # [configurations above]

    # for testing
    #  500 MB - 536870912
    #    1 GB - 1073741824
    #    2 GB - 2147483648
    #    3 GB - 3221225472
    capacity: 1073741824 # 1 GB bytes total blobber capacity
    read_price: 0.01     # token / GB for reading
    write_price: 1.00    # token / GB for writing
    # min_lock_demand is value in [0; 1] range; it represents number of tokens the
    # blobber earned even if a user will not read or write something
    # to an allocation; the number of tokens will be calculated by the following
    # formula
    #
    #     allocation_size * write_price * min_lock_demand
    #
    min_lock_demand: 0.1
    # max_offer_duration restrict long contacts where,
    # in the future, prices can be changed
    max_offer_duration: 744h # 31 day
    challenge_completion_time: 1m # 15m # duration to complete a challenge

    # delegate wallet for all rewards, if it's empty, then blobber ID used
    delegate_wallet: 'b145bf241eab00c9865a3551b18028a6d12b3ef84df8b4a5c317c8d184a82412'
    ```
4. Start blobbers (if not started yet) and export their IDs to use later.
    ```
    export BLOBBER1=f65af5d64000c7cd2883f4910eb69086f9d6e6635c744e62afcfab58b938ee25
    export BLOBBER2=7a90e6790bcd3d78422d7a230390edc102870fe58c15472073922024985b1c7d
    ```
    <!--
      export BLOBBER1=f65af5d64000c7cd2883f4910eb69086f9d6e6635c744e62afcfab58b938ee25
      export BLOBBER2=7a90e6790bcd3d78422d7a230390edc102870fe58c15472073922024985b1c7d
      export BLOBBER3=2f051ca6447d8712a020213672bece683dbd0d23a81fdf93ff273043a0764d18
      export BLOBBER4=2a4d5a5c6c0976873f426128d2ff23a060ee715bccf0fd3ca5e987d57f25b78e
    -->
5. Wait some time and make sure the blobbers has registered in the storage SC:
    ```
    ./zbox ls-blobbers
    ```
6. Add tokens to the stake pools of the blobber to allow them to accept
   allocations.
    ```
    ./zbox sp-lock --blobber_id $BLOBBER1 --tokens 2
    ./zbox sp-lock --blobber_id $BLOBBER2 --tokens 2
    ```
7. Check out stake pools of the blobbers. They should contains required stake.
    ```
    ./zbox sp-info --blobber_id $BLOBBER1
    ./zbox sp-info --blobber_id $BLOBBER2
    ```
    For example
    ```
    pool_id: a7782f5a4a68a242c8dbb163a72ae1f65d3db8b35e017439b265edcf969935a1
    balance: 12
    capacity:
      free:        120.0 GiB (for current write price)
      capacity:    1.0 GiB (blobber bid)
      write_price: 0.1 (blobber write price)
    offers: no opened offers
    delegate_pools:
    - id:          860703e3c21b6b0e60a00894ea5ac8d4150874118f5462995cce80d0e9510264
      balance:     10
      delegate_id: b145bf241eab00c9865a3551b18028a6d12b3ef84df8b4a5c317c8d184a82412
      earnings:    2 (payed interests for the delegate pool)
      penalty:     0 (penalty for the delegate pool)
      interests:   2 (interests not payed yet, can be given by 'sp-pay-interests' command)
    - id:          f50cf63bed2c89c5b4328bd429a93d087a354a6d8b792aa237ee760bcd003236
      balance:     1
      delegate_id: b145bf241eab00c9865a3551b18028a6d12b3ef84df8b4a5c317c8d184a82412
      earnings:    0.1 (payed interests for the delegate pool)
      penalty:     0 (penalty for the delegate pool)
      interests:   0.1 (interests not payed yet, can be given by 'sp-pay-interests' command)
    - id:          f68d7aa17e649cbb65c5c6a021d06b3021f88b6301adc556add5ad16b719fc11
      balance:     1
      delegate_id: b145bf241eab00c9865a3551b18028a6d12b3ef84df8b4a5c317c8d184a82412
      earnings:    0 (payed interests for the delegate pool)
      penalty:     0 (penalty for the delegate pool)
      interests:   0.1 (interests not payed yet, can be given by 'sp-pay-interests' command)
    earnings: 2.1 (total interests earnings for all delegate pools for all time)
    penalty: 0 (total blobber penalty for all time)
    rewards: (excluding interests)
      balance:   0 (current rewards can be unlocked)
      blobber:   0 (for all time)
      validator: 0 (for all time)
    ```
    This fields
    ```
    free:        120.0 GiB (for current write price)
    capacity:    1.0 GiB (blobber bid)
    write_price: 0.1 (blobber write price)
    ```
    The 'free' is staked capacity for current write price. A blobber can change
    its write price anytime and it will be changed. Even if the 'free' capacity
    is 100GB a blobber can't allocate more then its capacity bid. For example,
    if a blobber has zero write_price it can't allocate infinity. Thus a blobber
    can provide `min (free, bid)` for current time and terms.
8. Create and fund two new allocations.
    ```
    ./zbox newallocation --read_price 0.001-10 --write_price 0.01-10 --size 104857600 --lock 2 --data 1 --parity 1 --expire 48h
    ./zbox newallocation --read_price 0.001-10 --write_price 0.01-10 --size 104857600 --lock 2 --data 1 --parity 1 --expire 48h
    ```
    Export their IDs to use later.
    ```
    export ALLOC1=<put allocation 1 ID here>
    export ALLOC2=<put allocation 2 ID here>
    ```
9. Check out user's allocations list.
    ```
    ./zbox listallocations
    ```
10. Check out allocations independently.
    ```
    ./zbox get --allocation $ALLOC1
    ./zbox get --allocation $ALLOC2
    ```
11. Check out stake pools again, that should have offers for these allocations
    ```
    ./zbox sp-info --blobber_id $BLOBBER1
    ./zbox sp-info --blobber_id $BLOBBER2
    ```
12. Check out write pools of the allocations.
    ```
    ./zbox wp-info --allocation $ALLOC1
    ./zbox wp-info --allocation $ALLOC2
    ```
13. Update the first allocation, increasing its size. We don't provide more
    tokens, since related write pool already has enough tokens for the updating.
    ```
    ./zbox updateallocation --allocation $ALLOC1 --size 209715200
    ```
14. Check out its write pool again. Shouldn't be changed.
    ```
    ./zbox wp-info --allocation $ALLOC1
    ```
15. Check out blobbers offers again.
    ```
    ./zbox sp-info --blobber_id $BLOBBER1
    ./zbox sp-info --blobber_id $BLOBBER2
    ```
16. Generate random file to upload
    ```
    head -c 20M < /dev/urandom > random.bin
    ```
17. Upload it to the first allocations.
    ```
    ./zbox upload \
      --allocation $ALLOC1 \
      --commit \
      --localpath=random.bin \
      --remotepath=/remote/random.bin
    ```
18. Check out uploaded list
    ```
    ./zbox list --allocation $ALLOC1 --remotepath /remote
    ```
19. Check out related challenge and write pools after blobbers commit their
    write markers in SC.
    ```
    ./zbox cp-info --allocation $ALLOC1
    ./zbox wp-info --allocation $ALLOC1
    ```
20. Wait a challenge some time. Check challenge pool again.
21. Check out blobbers stake pools to see filling with rewards
    ```
    ./zbox sp-info --blobber_id $BLOBBER1
    ./zbox sp-info --blobber_id $BLOBBER2
    ```
22. Delete the file
    ```
    ./zbox delete --allocation $ALLOC1 --remotepath /remote/random.bin
    ```
23. Generate and upload another file.
    ```
    head -c 50M < /dev/urandom > random.bin
    ./zbox upload \
      --allocation $ALLOC1 \
      --commit \
      --localpath=random.bin \
      --remotepath=/remote/random.bin
    ```
24. Check out uploaded list
    ```
    ./zbox list --allocation $ALLOC1 --remotepath /remote
    ```
25. Check out related challenge and write pool after blobbers commit their
    write markers in SC.
    ```
    ./zbox cp-info --allocation $ALLOC1
    ./zbox wp-info --allocation $ALLOC1
    ```
26. Commit some tokens to a read pool.
    ```
    ./zbox rp-lock --allocation $ALLOC1 --duration 1h --tokens 1
    ```
27. Check out locked tokens in the read pool.
    ```
    ./zbox rp-info
    ```
28. Download the file.
    ```
    rm -f got.bin
    ./zbox download --allocation $ALLOC1 --localpath=got.bin \
        --remotepath /remote/random.bin
    ```
29. Make the allocation expired.
    ```
    ./zbox updateallocation --allocation $ALLOC1 --expiry -48h
    ./zbox get --allocation $ALLOC1
    ```
    Wait 'Challenge Completion Time' from response and finalize the allocation.
    ```
    ./zbox alloc-fini --allocation $ALLOC1
    ```
    And check all related pools (excluding read pool)
    ```
    ./zbox cp-info --allocation $ALLOC1
    ./zbox wp-info --allocation $ALLOC1
    ```
    Blobbers finalizes allocations automatically by some interval (see blobber
    configurations 'update_allocations_interval'). And in this case,
    the alloc-fini can fail with 'allocation already finalized'.
30. Cancel second allocation.
    ```
    head -c 50M < /dev/urandom > random.bin
    ./zbox upload                        \
        --allocation $ALLOC2             \
        --commit                         \
        --localpath=random.bin           \
        --remotepath=/remote/random.bin
    ```
    Then shutdown blobbers to make them fail their challenges for the new
    uploaded file. It's not 100% guaranteed that the blobber receive a challenge
    requests, since challenges generation based on some randomization.
    Wait challenge completion time (use `./zbox get --allocation $ALLOC2`).
    All generated challenges will be failed (due to time). Check out allocation
    object.
    ```
    ./zbox get --allocation $ALLOC2
    ```
    Make sure `total challenges` reaches `failed_challenges_to_cancel`
    configured in sc.yaml ('stroagesc'). Since, blobbers is down after the
    challenge_completion_time all the challenges are failed. But the
    filed_challenges field will be zero, because blobber doesn't send a
    challenge (the failed_challenges is where a blobber sends a failed
    challenge, there is mechanism for challenges expired for a case blobber is
    down).
    ```
    ./zbox alloc-cancel --allocation $ALLOC2
    ```
    Now
    ```
    ./zbox get --allocation $ALLOC2
    ```
    should show
    ```
    finalized:                 true
    canceled:                  true
    ```
31. Unlock read pool tokens
    ```
    ./zbox rp-info
    ```
    Use pool id in next command, but, make sure the pool has expired, wait
    otherwise. It's duration is 1 hour. The min allowed duration can be
    configured for BC in sc.yaml 'stroaagesc'.
    ```
    ./zbox rp-unlock --pool_id <POOL_ID>
    ```
32. Check out blobbers. Should not have allocated space.
    ```
    ./zbox ls-blobbers
    ```
33. Check out your wallet balance
    ```
    ./zwallet getbalance
    ```
34. Now transfer the rewards from your stakepool to your wallet.
    ```
    ./zbox collect-rewards --poolod $POOL_ID --provider_type blobber
    ```
35. You can now check your balance with
     ```
    ./zwallet getbalance
    ```
    unfortunately this might not show your rewards, as `getbalance` ony gives
your balance to three decimal places.


# Client specific API

Use `./zbox sp-user-info` to get all stake pools of current user.
