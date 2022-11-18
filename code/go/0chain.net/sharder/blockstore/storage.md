# Introduction

Sharder storage is simply the utilization of multiple partitions and mixture of storage types. It is able to use multiple partitions, s3-compatible storage and stores block's metadata in a rocksdb so that searching block is a trivial task. A simple cache feature is also provided so that recently accessed blocks are quickly available for next block's request.

Previous storage mechanism would use block hash to calculate path for the block. The hash was divided into 4 segments. Hash size is 64 characters so the path for a block would be: /base_path/hash[:3]/hash[3:6]/hash[6:9]/ where hash[9:] is the name of the file that stores block information.

This use of hash will create maximum of 16^9 = 2^36 => 68 billion directories. In a year we generate around 15 millions blocks. At this rate it will take 4358 years to make total usage of 68 billion directories. So in worst case it will be generating 3 new directories for each block and with block itself it will require 4 inodes to store single block. Since inodes are limited resource and the current block storage is limited in single partition, we should discard the current block storage mechanism.

# Implementation

We use mixture of 3 storage types:
1. Cache storage
2. Disk(HDD) storage
3. Minio(cold) storage

Cache is assigned number 1, disk is assigned number 2 and cold is assigned number 4. There can be mixture of storage types and Disk storage(2) must be included. So there are four mixtures:
1. DiskOnly         2          --> There can be multiple disks.
2. CacheAndDisk     1+2=3      --> It will contain a cache storage(basically SSD) and multiple disk.
3. DiskAndCold      2+4=6      --> It can contain multiple cold storages and multple HDDs. The block older than certain time interval will move from HDDs to cold storage. 
4. CacheDiskAndCold 1+2+4=7    --> It will contain a mix of Cache, Disk and Cold storages.

_Note that Cache storage will contain uncompressed data while Disk and Cold Storage will contain compressed data_

There are four strategies that can be set up for disk management.
1. Random --> It will choose active disk randomly.
2. Round Robin --> It will choose active disk in round robin fasion.
3. Min Size First -->  It will choose active disk that has stored blocks size lesser than other disks.
4. Min Count First --> It will choose active disk that has stored blocks lesser than other disks.
5. Fill First --> It will fill the disk first before moving to other active disk. This is yet to be implemented.

## Parameters for the cache config
path: Mounted path of the cache
size: Size of the cache. This parameter takes integer value and unit is in bytes. After the cache is filled LRU replacement policy is used to remove blocks from the cache. This value cannot be lesser than 500 MB, otherwise it will panic.

## Parameters for the disk config
path: Mounted path of the disk

size_to_maintain: This parameter takes integer value and the unit is in GB. Sharder will stop storing blocks in the respective disk if its available size is less than or equal to `size_to_maintain`. Defaults to 0 which means that it will store blocks until the disk is full.

inodes_to_maintain: This parameter takes integer value and unit is percentage. Sharder will stop storing blocks in the respective disk if its available inodes are less than or equal to `inodes_to_maintain%`. Defaults to 0, which means that it will store blocks until there are no inodes left.

allowed_block_numbers: This parameter takes integer value. Sharder will stop storing blocks in the respective disk if it is currently storing blocks greater than or equal to `allowed_block_numbers`. The disk will be active if its blocks are moved to cold storage. Defaults to 0, which means that it will store as much blocks as it can.

allowed_block_size: This parameter takes integer value and the unit is in GB. Sharder will stop storing blocks in the respective disk if it is currently storing blocks greater than or equal to `allowed_block_size`. The disk will be active if its blocks are moved to cold storage. Defaults to 0, which means that it will store as much blocks as it can.

## Parameters for the cold storage
Any s3-compatible or minio-compatible storage will work as cold storage. Like disk, there can be multiple cold storages associated with the sharder. For example, one can deploy minio with cheapest storage(like magnetic tape, etc.) as a cold storage. The parameters are:

delete_local: This parameter takes boolean value. Sharder will delete the block in disk if it successfully moved it to the cold storage. Defaults to false.

storage_service_url: This is the domain name of the server. For aws s3 it is `s3.amazonaws.com`.

access_id: Access ID of the service.

secret_access_key: Secret Access key of the service.

bucket_name: Bucket name where the blocks will be moved to.

allowed_block_numbers: Same as above.

allowed_block_size: Same as above.

## How is block stored inside a disk?
Note that we cannot store all the blocks inside a directory and expect it to be faster. It will take time to do lookup of a file in a directory where there are huge number of files. So we should limit number of blocks in a directory. Currently it is limited to 2000 blocks in a directory. We can find the best number later on.

Consider a mounted path of a disk if `/dir`. A `blocks` directory will be created inside it. So the base path for the blocks will be `/dir/blocks`. Sharder will create 2000 directories further with name `K0, K1,...., K1999`. Each `K` directory will contain 2000 directories with name `0,1,...,1999`. Each directory will then contain 2000 blocks.
So a disk can have maximum of 8*10^9 blocks = 8 billion blocks. It will take around 500 years to generate 8 billion blocks.

This constant of 2000 is hardcoded. Sharder can simply change the constant value and it will just work fine.

The blocks will start to move from `/dir/blocks/K0/0` directory. So once all the directories are filled it will check if initial `/dir/blocks/K0/0` directory is available. So the block storage inside is disk is circular.

__Note: The directories are not created pre-hand. They are created only when required. But once created they are not deleted because it will be used later on__

## Metadata
The metadata of a block is stored in rocksdb. It stores block's hash, either file path of block on disk or cold storage or both depending on storage type and in which storage is block stored(it can be disk or cold or both(if delete_local=false)). Rocksdb is preferred because it is scalable and has high read and write performance. When a block is requested, sharder checks for its metadata in rocksdb and it will read block from the disk or cold storage depending on whereabout of block in the metadata. If cache is enabled, it searches for the block directly into the cache before even searching rocksdb.

# TODO
We can extend storages on the fly. We should have some IPC to let sharder know that a storage is added and it should consider adding this storage in current active list of storages.