STORAGE SC
==========

# Flow

## Blobber

### 1. Blobber: send transaction -> registration

Provide

 - capacity                   bytes
 - min_lock_demand            [0; 1]
 - write_price                tok / GB
 - read_price                 tok / read
 - max_offer_duration         time.Duration
 - challenge_completion_time  time.Duration

 - number of tokens to lock

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


# Setup

## Order

 1. clean blockchain
 2. clean blobbers
 3 .rebuild all applications: zboxcli, zwallet, blobbers, sharder, miners
 4. start sharder
 5. start miners
 6. remove wallet.json and allocation.txt
 7. configure blobbers (remember capacity, write_price, id)
 8. create new wallet (zwallet)
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

1. start sharder, start miners
2. execute
```bash
for run in {1..10}
do
    ./zwallet faucet --methodName pour --input “{Pay day}”
done
./zwallet send --to_client_id f65af5d64000c7cd2883f4910eb69086f9d6e6635c744e62afcfab58b938ee25 --token 2 --desc "to register"
./zwallet send --to_client_id 7a90e6790bcd3d78422d7a230390edc102870fe58c15472073922024985b1c7d --token 2 --desc "to register"
```
3. start blobbers, wait their registration
```
./zwallet getblobbers
```
4. check blobber's stake pools
```
./zwallet getstakelockedtokens --blobber_id f65af5d64000c7cd2883f4910eb69086f9d6e6635c744e62afcfab58b938ee25
./zwallet getstakelockedtokens --blobber_id 7a90e6790bcd3d78422d7a230390edc102870fe58c15472073922024985b1c7d
```
5. create new allocation
```
./zbox newallocation --read_price 0.001-10 --write_price 0.01-10 --size 104857600 --lock 2 --data 1 --parity 1 --expire 48h
```
```
46ec6678df10e1808616375b4eb51317689700ecec7333ebd606fb0935081135 (allocation id for example)
```
Let's export it and use in next requests
```
export ALLOC="46ec6678df10e1808616375b4eb51317689700ecec7333ebd606fb0935081135"
```
7. check out stake pools again
```
./zwallet getstakelockedtokens --blobber_id f65af5d64000c7cd2883f4910eb69086f9d6e6635c744e62afcfab58b938ee25
./zwallet getstakelockedtokens --blobber_id 7a90e6790bcd3d78422d7a230390edc102870fe58c15472073922024985b1c7d
```
8. check out write pool of the allocation
```
./zwallet getwritelockedtokens --allocation_id $ALLOC
```
9. update allocation (increase size to 200MB, don't provide tokens, since we already have enough in the write pool)
```
./zbox updateallocation --allocation $ALLOC --size 209715200
```
10. check out pools and allocation
11. if blobber reduces its capacity next registration, then some tokens becomes
unlocked, and can be moved to blobber wallet by its owner; for example
blobber1.json:
```json
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
```
```
./zwallet --wallet blobber1.json stakeunlock
```
12. generate a random file to upload
```
head -c 20M < /dev/urandom > random.bin
```
13. upload the file to blobbers
```
./zbox upload \
    --allocation $ALLOC \
    --commit \
    --localpath=random.bin \
    --remotepath=/remote/random.bin
```
14. check out the file
```
./zbox list --allocation $ALLOC --remotepath /remote/
```
15. wait some time and make sure tokens from challenge pool moves to
blobber's stake pool (unlocked)
```
./zwallet getchallengelockedtokens --allocation_id $ALLOC
./zwallet getstakelockedtokens --blobber_id f65af5d64000c7cd2883f4910eb69086f9d6e6635c744e62afcfab58b938ee25
```
