Vesting SC
==========

Vesting SC moves locked tokens to desired destinations. The movement
depends on the time elapsed from the beginning of the lock.


# Demo

1. Create wallet and fill it with tokens

```
for run in {1..10}
do
    ./zwallet faucet --methodName pour --input “{Pay day}”
done
```

2. Create two more wallet as destinations

```
./zwallet --wallet one.json getbalance
./zwallet --wallet two.json getbalance
```

3. Check out IDs of the destinations.

```
cat ~/.zcn/one.json
cat ~/.zcn.two.json
```
"client_id" is the wanted ID. Let's export them to use
```
export DST1=<client_id from one.json>
export DST1=<client_id from two.json>
```

4. Check out vesting pool configurations.

```
./zwallet vp-config
```

5. Create new vesting pool.

```
./zwallet vp-add \
    --description "for testing"
    --duration 5m
    --lock 5
    --d $DST1:1
    --d $DST2:2
```

E.g. after 5 minutes DST1 will have 1 token `($DST1:1)` and DST2 will have 2
tokens from the pool.

6. Check out all pools created by current client.

```
./zwallet vp-list
```

Let's export ID of the pool as POOL.

```
export POOL=<id from the list>
```

7. Check out pool information.

```
./zwallet vp-info --pool_id $POOL
```

For example:

```yaml
pool_id:      ...long pool id omitted...
balance:      5
can unlock:   4.3800000001
description:  for testing
start_time:   2020-04-16 15:34:06 +0400 +04
expire_at:    2020-04-16 15:39:06 +0400 +04
destinations:
  - id:          d9d79f931add44afcaf534b9595fe01ffc6b5ffbccbd142d5160c1f68414024a
    vesting:     1
    can unlock:  0.2066666666
    last unlock: 2020-04-16 15:34:06 +0400 +04
  - id:          993512b83ccc042d0766240b25325dcf8e871858aaa8539e613afb44631a3fbf
    vesting:     2
    can unlock:  0.4133333333
    last unlock: 2020-04-16 15:34:06 +0400 +04
client_id:    29a63a9b70d129ae2c39dc65c411bfa84c58df6b8ad78993c085f3e08e5dd503
```

The 'balance' is technical balance, part of which belongs to destinations.

8. Trigger token movements by the owner.

```
./zwallet vp-trigger --pool_id $POOL
```
Check out info, check out destinations' balances.
```
./zwallet vp-info --pool_id $POOL
./zwallet --wallet one.json getbalance
./zwallet --wallet two.json getbalance
```

9. Unlock part of tokens as a destination.

```
./zwallet --wallet one.json vp-unlock --pool_id $POOL
```
And check out its balance next
```
./zwallet --wallet one.json getbalance
```

10. Unlock as owner. Stop vesting.

```
./zwallet vp-trigger --pool_id $POOL
```

Check out balance.

11. Lock more tokens to the pool.

Also, it's possible to add tokens to the pool

```
./zwallet vp-lock --pool_id $POOL --lock 1.1
```

12. Delete a pool.

```
./zwallet vp-delete --pool_id $POOL
```

It moves all vested tokens to destinations. And all left tokens to the owner.
