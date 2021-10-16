## Inode Management
There are limited number of inodes which limits the number of files that a filesystem can hold. Default inode to size ratio is **1:16KB**. If we are going to store smaller files less than 16KB we might want to increase inode numbers so that filesystem can hold large number of files.
*Note: Inode too occupy space*

Command to create filesystem with custom inode numbers
`sudo mkfs.[fs] -i [size] [partition]`
`eg: sudo mkfs.ext4 -i 8192 /dev/fs1`
*Note: Its better to pass -T flag for above command as it filesystem can optimize space allocation*
