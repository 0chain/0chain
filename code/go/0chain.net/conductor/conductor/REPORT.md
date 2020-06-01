<!--

REPORT
======

# Terms

VC - View Change
MB - Magic Block

# Outside miner comes up on phase X.

A miner not registered.

1. Miner comes up on 'start' and joins MB next VC.
2. Miner comes up on any other phase and doesn't join MB
   next VC, but joins MB on VC after the next VC.

Nothing is stuck.

# Miner goes down on phase X and doesn't come up on the same VC.

For phases:
- start
- contribute
- share
- publish

the miner leaves MB next VC.

For phase _wait_ the miner leaves MB on VC after next VC.

Nothing is stuck.

# Miner goes down in phase X and comes up in phase Y

#### Miner goes down on 'start' and comes up shortly in the 'start'

Nothing happens.

#### Miner goes down on 'start' and comes up phase 'contribute'

Nothing happens.

#### Miner goes down on 'contribute' and comes up phase 'share'

In this case the miner leaves MB after VC.

#### Miner goes down on 'share' and comes up in phase 'start' of next VC

	TODO: WIP

# Artificially force x miners to only send y miners a sign for the signOrShare

	TODO: WIP

# Artificially force x miners' si to be revealed

	TODO: unknown

# Artificially force x miners to send bad share to y miners.

	TODO: WIP

-->
