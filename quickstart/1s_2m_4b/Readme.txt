This set of a simple scripts is designed to quick run 1 sharder 2 miners 4 blobbers network 
on a local machine.
And to perform some basic testing.


--------------------------------------------------

Set paths in the base/paths.sh

Also You could copy the whole /quickstart directory to the arbitrary folder.

--------------------------------------------------

Then run:
base/git_pull_all.sh

It will download master branches, 
and perform a reset to the commits that these scripts were tested with.

0Chain:     ac6c2253
0dns:       d6e1dbb3
blobber:    28c31930
gosdk:      54902b25
zbox:       25f8af1a
zwallet:    815813a2

--------------------------------------------------

The /reproduce directory is for instructions to reproduce a bug or something.


--------------------------------------------------
Configs management


Run config/compare_wdirs.sh to make sure that all docker.local folders
was not changed.

> If they are equal, you can run config/patch.sh
! If they are not, you have to figure out what was changed.


To see differences between config/reference and config/1s_2m_4b run
config/compare_patch.sh


--------------------------------------------------


The time delays in the reliable/restart_from_scratch.sh
are very important.
Try to adjust it for your CPU speed.

--------------------------------------------------


After patching run 

> reliable/restart_from_scratch.sh

Then if you have a google-chrome installed then
run the reliable/diagnostics_html.sh to make sure that the network is up


Then run reliable/create_allocation.sh


Then run reliable/upload_file_to_current_alloc.sh

It will upload zbox and zwalletcli bineries from 
base/paths.sh, just for test.


--------------------------------------------------


To rebuild and restart miners, run base/rebuild_miners.sh
To rebuild and restart sharders, run base/rebuild_sharders.sh
etc...

To see the logs, run /base/logs_miners.sh Error
the "Error" is a string to search in logs.
etc...



--------------------------------------------------
To clear all - run base/clear_all.sh
etc..

--------------------------------------------------


In the 0Chain/docker.local/config/b0mnode1_keys.txt file
the first line is a publik_key for the node
and the second line is
private key similar to wallet.json - can verify it with keygen repo (its public)
For non-genesis miners that join such as b0mnode5_keys.txt, 
the additional items are client_id, hostname, n2n_IP, https path.
