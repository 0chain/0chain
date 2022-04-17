# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Add settings update cool down to prevent DDoS attack #1008
- Implement transaction cost #1006
- SC endpoint: `/writemarkers` #964, please update if there are new APIs added
- Implement transaction errors #930
- Add events db for RESTful APIs

### Changed
- Replace MPT data serialization package from JSON to msgp(msgpack) #1003
- Rewrite block blobber reward and improve performance by replacing big lists with partitions #963
- Remove interest from miners and sharders #948

### Fixed
- Stabilize the large network via PR #849
- Fix the chain stuck on large network via PR #682
- Fix all lint errors #994
- Closed issues: #1074, #1073, #1060, #1047, #1027, #1022, #1011, #993, // please update the list for notable fixes.
- Closed conductor issues: #989, #988, #987, #986, #985, #984, #983, #981, #980, #979, #978, #971, #965, #953, #950, #931, #918


### Removed
// Please update if anything is removed, especially APIs,

[Unreleased] https://github.com/0chain/0chain/compare/master...staging
[0.20.9] https://github.com/0chain/0chain/compare/0.20.3...0.20.9
[0.20.3] https://github.com/0chain/0chain/compare/0.20.2...0.20.3
[0.20.2] https://github.com/0chain/0chain/compare/0.20.1...0.20.2
[0.20.1] https://github.com/0chain/0chain/compare/0.20.0...0.20.1
[0.20.0] https://github.com/0chain/0chain/compare/test-0.0.1...0.20.0
[test-0.01] https://github.com/0chain/0chain/compare/testnet-v0.9...test-0.0.1
[testnet-v0.9] https://github.com/0chain/0chain/compare/testnet-v0.8...testnet-v0.9
[testnet-v0.8] https://github.com/0chain/0chain/compare/testnet-v0.7...testnet-v0.8
[testnet-v0.7] https://github.com/0chain/0chain/compare/testnet-v0.6...testnet-v0.7
[testnet-v0.6] https://github.com/0chain/0chain/compare/testnet-v0.5...testnet-v0.6
[testnet-v0.5] https://github.com/0chain/0chain/compare/testnet-v0.4...testnet-v0.5
[testnet-v0.4] https://github.com/0chain/0chain/compare/testnet-v0.3...testnet-v0.4
[testnet-v0.3] https://github.com/0chain/0chain/compare/testnet-v0.2...testnet-v0.3
[testnet-v0.2] https://github.com/0chain/0chain/compare/testnet-v0.1...testnet-v0.2
[testnet-v0.1] https://github.com/0chain/0chain/tree/testnet-v0.1
