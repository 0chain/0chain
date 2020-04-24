STORAGE SC
==========

# SC testing

1. Enter the SC directory (directory with this README)
2. Execute unit-tests
    ```
    go test -cover -coverprofile=cover.out && go tool cover -html=cover.out -o=cover.html
    ```
3. Open generated cover.html file to see tests coverage.

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

 - number of tokens to lock


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
10. send (capacity * write_price) tokens to blobbers to allow them register

	- blobber 1 id: f65af5d64000c7cd2883f4910eb69086f9d6e6635c744e62afcfab58b938ee25
	- blobber 2 id: 7a90e6790bcd3d78422d7a230390edc102870fe58c15472073922024985b1c7d

	./zwallet send --to_client_id f65af5d64000c7cd2883f4910eb69086f9d6e6635c744e62afcfab58b938ee25 --token 2 --desc "to register"
	./zwallet send --to_client_id 7a90e6790bcd3d78422d7a230390edc102870fe58c15472073922024985b1c7d --token 2 --desc "to register"

11. start blobbers, wait their registration
12. create new allocation providing tokens to write pool
13. check out write_pool
14. check out blobbers
15. check out blobbers stake pools

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
3. Send some tokens to blobbers to allow them to register. In the example we
   are using blobber1 and blobber2. Export blobber identifier first to simplify
   commands
    ```
    export BLOBBER1=f65af5d64000c7cd2883f4910eb69086f9d6e6635c744e62afcfab58b938ee25
    export BLOBBER2=7a90e6790bcd3d78422d7a230390edc102870fe58c15472073922024985b1c7d
    ```
    ```
    ./zwallet send --to_client_id $BLOBBER1 --token 2 --desc "to register"
    ./zwallet send --to_client_id $BLOBBER2 --token 2 --desc "to register"
    ```
4. Setup blobbers' and validators' wallets in `~/.zcn/` directory to use them
    later. We will use them to check out balance. Blobber/Validator 1.
    ```
    cat > ~/.zcn/blobber1.json << EOF
    {
      "client_id": "f65af5d64000c7cd2883f4910eb69086f9d6e6635c744e62afcfab58b938ee25",
      "client_key": "de52c0a51872d5d2ec04dbc15a6f0696cba22657b80520e1d070e72de64c9b04e19ce3223cae3c743a20184158457582ffe9c369ca9218c04bfe83a26a62d88d",
      "keys": [
        {
          "public_key": "de52c0a51872d5d2ec04dbc15a6f0696cba22657b80520e1d070e72de64c9b04e19ce3223cae3c743a20184158457582ffe9c369ca9218c04bfe83a26a62d88d",
          "private_key": "17fa2ab0fb49249cb46dbc13e4e9e6853af8b1506e48d84c03e5e92f6348bb1d"
        }
      ],
      "version": "1.0",
      "date_created": "2020-03-16 00:47:58.247961953 +0400 +04 m=+0.015793530"
    }
    EOF
    ```
    Blobber/Validator 2.
    ```
    cat > ~/.zcn/blobber2.json << EOF
    {
      "client_id": "7a90e6790bcd3d78422d7a230390edc102870fe58c15472073922024985b1c7d",
      "client_key": "e4dc5262ed8e20583e3293f358cc21aa77c2308ea773bab8913670ffeb5aa30d7e2effbce51f323b5b228ad01f71dc587b923e4aab7663a573ece5506f2e3b0e",
      "keys": [
        {
          "public_key": "e4dc5262ed8e20583e3293f358cc21aa77c2308ea773bab8913670ffeb5aa30d7e2effbce51f323b5b228ad01f71dc587b923e4aab7663a573ece5506f2e3b0e",
          "private_key": "4b6d9c5f7b0386e36b212324ea52f5ff17a9ed1338ca901d7f7fa7637159a912"
        }
      ],
      "version": "1.0",
      "date_created": "2020-03-16 00:47:58.247961953 +0400 +04 m=+0.015793530"
    }
    EOF
    ```
5. Start blobbers and validators. Wait their registration.
    A storage SC related blobber configurations used for this examples.
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

    # [configurations below]
    ```
    Check blobbers' registrations.
    ```
    ./zbox ls-blobbers
    ```
6. Check out stake pools of the blobbers. They should contains required stake.
    ```
    ./zbox sp-info --blobber_id $BLOBBER1
    ./zbox sp-info --blobber_id $BLOBBER2
    ```
7. Create and fund two new allocations.
    ```
    ./zbox newallocation --read_price 0.001-10 --write_price 0.01-10 --size 104857600 --lock 2 --data 1 --parity 1 --expire 48h
    ./zbox newallocation --read_price 0.001-10 --write_price 0.01-10 --size 104857600 --lock 2 --data 1 --parity 1 --expire 48h
    ```
    Export their IDs to use later.
    ```
    export ALLOC1=<put allocation 1 ID here>
    export ALLOC2=<put allocation 2 ID here>
    ```
8. Check out user's allocations list.
    ```
    ./zbox listallocations
    ```
9. Check out allocations independently.
    ```
    ./zbox get --allocation $ALLOC1
    ./zbox get --allocation $ALLOC2
    ```
10. Check out stake pools again, that should have offers for these allocations
    ```
    ./zbox sp-info --blobber_id $BLOBBER1
    ./zbox sp-info --blobber_id $BLOBBER2
    ```
11. Check out write pools of the allocations.
    ```
    ./zbox wp-info --allocation $ALLOC1
    ./zbox wp-info --allocation $ALLOC2
    ```
12. Update the first allocation, increasing its size. We don't provide more
    tokens, since related write pool already has enough tokens for the updating.
    ```
    ./zbox updateallocation --allocation $ALLOC1 --size 209715200
    ```
13. Check out its write pool again. Shouldn't be changed.
    ```
    ./zbox wp-info --allocation $ALLOC1
    ```
14. Check out blobbers offers again.
    ```
    ./zbox sp-info --blobber_id $BLOBBER1
    ./zbox sp-info --blobber_id $BLOBBER2
    ```
15. Generate random file to upload
    ```
    head -c 20M < /dev/urandom > random.bin
    ```
16. Upload it to the first allocations.
    ```
    ./zbox upload \
      --allocation $ALLOC1 \
      --commit \
      --localpath=random.bin \
      --remotepath=/remote/random.bin
    ```
17. Check out uploaded list
    ```
    ./zbox list --allocation $ALLOC1 --remotepath /remote
    ```
18. Check out related challenge and write pools after blobbers commit their
    write markers in SC.
    ```
    ./zbox cp-info --allocation $ALLOC1
    ./zbox wp-info --allocation $ALLOC1
    ```
19. Wait a challenge some time. Check challenge pool again.
20. Check out blobbers stake pools to see filling with rewards (OVERFILL column)
    ```
    ./zbox sp-info --blobber_id $BLOBBER1
    ./zbox sp-info --blobber_id $BLOBBER2
    ```
21. Delete the file
    ```
    ./zbox delete --allocation $ALLOC1 --remotepath /remote/random.bin
    ```
22. Check out challenge and write pools again.
    ```
    ./zbox cp-info --allocation $ALLOC1
    ./zbox wp-info --allocation $ALLOC1
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
    ./zbox rp-lock --allocation $ALLOC1 --duration 40m --tokens 1
    ```
27. Check out locked tokens in the read pool.
    ```
    ./zbox rp-info --allocation $ALLOC1
    ```
28. Download the file.
    ```
    rm -f got.bin
    ./zbox download --allocation $ALLOC1 --localpath=got.bin \
        --remotepath /remote/random.bin
    ```
30. Make the allocation expired.
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
31. Cancel second allocation.
    ```
    ./zbox alloc-cancel --allocation $ALLOC2
    ```
32. Unlock read pool tokens
    ```
    ./zbox rp-info
    ```
    Use pool id in next command
    ```
    ./zbox rp-unlock --pool_id <POOL_ID>
    ```
33. Check out blobbers. Should not have allocated space.
    ```
    ./zbox ls-blobbers
    ```
