# View Change

## Table of Contents
- [Phases](#phases)


### Phases

There are 5 phases for the view change: start, contribute, share, publish, and wait.

#### Start

During the start phase the smart contract will get the list of all miners who have registered. The list is sorted by stake and picks N miners to become the dkg miners. If there are more than N miners in the list the sort allows only those with the most staked to be part of the dkg miners. Once the list is made, it is saved in the smart contract's mpt. This allows miners who aren't part of the current view change to request the list from the sharders through a rest api. 

#### Contribute

Part of the dkg miners list is also the N (max amount of miners), K (threshold needed for dkg), T (threshold need for security). All miners in the dkg miners list use the N and T to create an MPK. This MPK is sent to the smart contract. Any miner who fails to send an MPK during the contribute period is taken out of the dkg miners list. This is done because the MPK will be used to verify the signs during the share phase and the share or signs during the publish phase.

#### Share
During this phase the miners only communicate with each other. They use the MPKs they sent to the blockchain to derive secret shares for every miner in the dkg miners list and send them the share. When a miner recieves a share they use the published MPK from the miner who sent the share to verify it. Once verified the miner will use the share to sign a message back to the original miner. The original miner collects the message and signature from all the other miners in the dkg miners list. If a miner doesn't recieve a signature they use the secret share instead for that miner. 

#### Publish
Every miner sends the collection of shares or signs to the smart contract. The smart contract verifies the share and signs are correct. If enough shares for one miner come in they are removed from the dkg miners list. Likewise, if a miner doesn't publish the shares or signs then they are also removed from the list.

#### Wait

At the beginning of this phase a magic block is created for the next view change. Every miner uses the list on the magic block to determine the secret shares used for their personal private key for the VRF in the next view change.