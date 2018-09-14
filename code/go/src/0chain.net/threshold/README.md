`0chain.net/threshold/model/` has the base cryptographic distributed key
generator (DKG) and verifiable random function (VRF) logic.

`0chain.net/threshold/protocol/` adds timing and retransmission logic necessary
for a practical implementation on a cluster of machines.

`0chain.net/threshold/network/` integrates the above into the
`0chain.net/node/` API.
