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
cat ~/.zcn/two.json
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
./zwallet vp-add                \
    --description "for testing" \
    --duration 5m               \
    --lock 5                    \
    --d $DST1:1                 \
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
pool_id:      2bba5b05949ea59c80aed3ac3474d7379d3be737e8eb5a968c52295e48333ead:vestingpool:d808792fb07faccdb6c4d6e944769d5387cb280d1a683960d76de1d9bef2c601
balance:      2.2633333335
can unlock:   0.6466666668
vested:       0.3833333333 (sent, real value)
pending:      1.6166666667 (not vested, real value)
description:  for testing
start_time:   2020-04-24 19:32:34 +0400 +04
expire_at:    2020-04-24 19:37:34 +0400 +04
destinations:
  - id:          8ea9488df3bd528405e184a8fed82ee104420c002732b03a34de7a827ae60aa3
    vesting:     1
    can unlock:  0.02
    vested:      0.3833333333 (sent, real value)
    pending:     0.6166666667 (not vested, real value)
    last unlock: 2020-04-24 19:34:29 +0400 +04
  - id:          029d707cb0070ae006c8fd728511a6e6184e1416bd4026a9da875f7c25c0fd1c
    vesting:     1
    can unlock:  0.4033333333
    vested:      0 (sent, real value)
    pending:     1 (not vested, real value)
    last unlock: 2020-04-24 19:32:34 +0400 +04
client_id:    66c7560d7ec4e74565b7ec91b00f4e3c06cf2859ef328cf28449ad44d68f8c68
```

- The 'balance' is technical balance, part of which belongs to destinations.
- The 'can unlock' is value can be unlocked by owner (the top), or a client
  (destinations).
- The 'vested' and 'pending' is real tokens moved or waiting to be moved.
- The 'last unlock' is last trigger or unlock (in case where client unlocks).

Tokens value for a trigger or an unlocking based on time period from the last
unlock and current time and entire period length. If value for period is
zero (rounded to zero), then it can't be unlocked. The trigger never fails
in this cases regarding other destinations (because trigger affects all
destinations).

If pool expired, then a trigger or an unlock moves all tokens not vested yet
to the destinations. But empty and expired pool triggering fails.

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

10. Unlock as owner all over required part.

```
./zwallet vp-unlock --pool_id $POOL
```

Check out balances.

11. Stop vesting to a destination.

```
./zwallet vp-stop --pool_id $POOL --d $DST2
```

Check out with `./zwallet vp-info --pool_id $POOL`

12. Delete a pool.

```
./zwallet vp-delete --pool_id $POOL
```

It moves all vested tokens to destinations. And all left tokens to the owner.
