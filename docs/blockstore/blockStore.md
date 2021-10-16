# Block Storage Description and Definition

## BMR(Block Meta Record)
A block is stored in some partition and after some time it is moved to cold storage (minio tiering/cold tiering). A frequently used block is also stored in SSD(uncompressed block). So when a block is requested, searching SSD and then HDD and then cold storage is very inefficient. To limit search, each block's meta record is stored in some key-value database(in SSD) which provides information if block is available within the sharder and if available then where.

*If SSD got corrupted then we can always re-create **BMR** from existing Warm and Cold Tier.*

Based on access speed we define following terms:
**Hot Tiering**
Block is in SSD

**Warm Tiering**
Block is in HDD

**Cold Tiering**
Block is in some minio api compatible server 

A block can be both in Hot Tiering and Warm Tiering or Hot Tiering and Cold Tiering but it can never be in both Warm Tiering and Cold Tiering. Hot Tier is used as a cache based of access frequency and block generation date.


## Strategies
Which strategy to use will be provided in config file.

### Min Size First
This strategy selects partition based on minimum size occupied by files i.e. partition that has maximum available storage capacity will be selected.

### Min Count First
This strategy selects partition based on minimum number of blocks stored i.e. partition that has least number of blocks stored will be selected.

### Round Robin
This strategy selects partition sequentially i.e. each partition gets turn to store block.

### Random
This strategy selects partition randomly.

*Note: If any partition is unable to store block then it will be removed from partition list and next partition will be selected*


## Directory Management
There can be multiple mounted volumes(partitions). Each mounted volumes should be listed in config file before starting sharder.
Number of inodes are defaulted to 1:16KB which can be changed as per requirement. But if on average block size if 1 MB or greater than 16KB then we don't need to configure inode number but should atleast confirm it with file system we are using(as file system can have their own inode to size ratio).

To store blocks in Warm Tier, directory names are defined:
    `Kilo(K) --> Contains 1000 directories that contains 1000 blocks each so 10^6 blocks
	 Mega(M) --> Contains 1000 K directories so each M directory contains 10^9 blocks.
	 Giga(G) --> Contains 1000 M directories so each G directory contains 10 ^12 blocks.
	 Peta(P) --> Contains 1000 G directories so each P directory contains 10^15 blocks.
	 Exa (E) --> Contains 1000 P directories so each E directory contains 10^18 blocks.
	 Zillion(Z) --> Contains 1000 E directories so each Z directory contains 10^21 blocks.
    `
With each directory containing 1000's of 1MB sized blocks, 1000 kilo directories contains around 1PB of data which is more than enough for a single partition. Directory is created on demand i.e. it is not pre-created.

Gradually older blocks will also be moved to cold tier thus creating size for new blocks.


## Caching
To provide fastest access to frequently accessed block, it is copied into SSD(hot tier) uncompressed. As BMR lets us know if block is in SSD or not, we can increase block access performance.