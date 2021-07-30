```puml
title block rewards smart contract run every block
boundary chain
control storagesc
database MPT
chain -> storagesc : pay_blobber_block_rewards
    MPT -> storagesc : config (sc.yaml.storagesc) 
    MPT -> storagesc : total stakes, all blobbers\nmap[blobberIds]state.Balance 
    storagesc -> storagesc : calculate rewards
    MPT -> storagesc : pending mints\nmap[blobberId]float64
    storagesc -> storagesc : add rewards to pending mints
    alt max mint exceeded
    storagesc -> chain : error\nmax mint exceeded
    end
    storagesc -> MPT : save pending mints
    storagesc -> MPT : sace config (MaxMint)   
storagesc -> chain
```

```puml
title changes to storagesc.getStakePool
control getStakePool
database MPT
MPT ->  getStakePool : stake pool
MPT -> getStakePool : pending mints\nmap[blobberId]float64
alt pending mit for this stake pool >= 1
    getStakePool -> getStakePool : split pending mint between stake holders
    getStakePool -> MPT : mint tokens for this stake pool
    getStakePool -> getStakePool : decrement pending mints
    getStakePool -> MPT : save pending mitns\nif tokens minted
end 
getStakePool -> getStakePool : local save total stake
```

```puml
title chainges to stakePool.save
control save
database MPT
save -> MPT : save stake pool
alt total stake has changed
MPT -> save : total stakes, all blobbers\nmap[blobberIds]state.Balance 
save -> save : update total stakes this blobber
save -> MPT : save total stakes, all blobbers
end
```