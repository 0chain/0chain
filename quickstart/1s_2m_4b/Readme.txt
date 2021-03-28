This set of a simple scripts is designed to standardize workflows during the development process 
on a local machine.


--------------------------------------------------

Unpack the zworkflows_1s_2m_4b.tar.gz to the arbitrary directory.

WARNING:

Set paths in the base/paths.sh


--------------------------------------------------


WARNING:

0Chain:     master: 394c1f8c
0dns:       master: d6e1dbb3
blobber:    master: ca6a8a3b
gosdk:      master: 1ce76cf7
zboxcli:    master: f9802b7d
zwalletcli: master: 815813a2

--------------------------------------------------

WARNING:

The time delays in the reliable/restart_from_scratch.sh
are very important.
Try to adjust it for your CPU speed.


--------------------------------------------------

The /reproduce directory is for instructions to reproduce a bug or something.


--------------------------------------------------
Configs management

Check out current master branches: 0Chain, 0dns, blobber

WARNING:

Run config/compare_reference.sh to make sure that all docker.local folders
was not changed.

> If they are equal, you can run config/patch.sh
! If they are not, you have to figure out what was changed.


--------------------------------------------------

run: base/rebuild_zbox.sh
run: base?rebuild_zwallet.sh

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
