ZCNSC SC
==========

Deployments
===================
- Ropsten
- Goerli
- Polygon/Matic

## Ropsten
0Chain WToken **(NCNW)**    
Max Total Supply:  
100,000,000 NCNW

Code verified

### Owner

0x860FA46F170a87dF44D7bB867AA4a5D2813127c1  
Owner2: 0x96b6aF9396Eb48B7F4AFD4181c04d95382df5dD6  
Owner3: 0xfcb57642c15d2251cd7a2fd0eb692e6f8bd95874  

#### Summary:

https://ropsten.etherscan.io/token/0x930E1BE76461587969Cb7eB9BFe61166b1E70244

| Contract | Address     |
|----------|-------|
| Token   | 0x930E1BE76461587969Cb7eB9BFe61166b1E70244 |
| Authorizer | 0xFE20Ce9fBe514397427d20C91CB657a4478A0FFa  |
| Bridge | 0xF26B52df8c6D9b9C20bfD7819Bed75a75258c7dB  |


## Goerli
Code verified

0Chain Token **(ZCN)**  
Max Total Supply:  
500,000,000 ZCN

### Owner

0x860FA46F170a87dF44D7bB867AA4a5D2813127c1

#### Summary:

https://goerli.etherscan.io/token/0xA839639D94655C4a42CBb3070242f0947C80CeFB

| Contract | Address     |
|----------|-------|
| Token   | 0xA839639D94655C4a42CBb3070242f0947C80CeFB |
| Authorizer | 0xd37F574927f88566Fe9AE8623Fbb5F85a96b2DdB  |
| Bridge | 0x0aCF4697B058E2Cd5B99a1f1542B75b7319e08Cf  |


## Polygon
Code verified

0Chain Token **(ZCN)**  
Total Supply:  
500,000,000 ZCN

#### Summary:

https://mumbai.polygonscan.com/token/0x5ce9b8725c64b4e31491811c229ea2fe6545d4b0
https://mumbai.polygonscan.com/address/0xA839639D94655C4a42CBb3070242f0947C80CeFB
https://mumbai.polygonscan.com/address/0xd37F574927f88566Fe9AE8623Fbb5F85a96b2DdB


| Contract | Address     |
|----------|-------|
| Token   | 0x5ce9b8725c64b4e31491811c229ea2fe6545d4b0 |
| Authorizer | 0xA839639D94655C4a42CBb3070242f0947C80CeFB  |
| Bridge | 0xd37F574927f88566Fe9AE8623Fbb5F85a96b2DdB  |

API Documentation
=============

ZCNSC SC converts token from `native ZCN token` network to `Ethereum wrapped WZCN token`
and in reverse order converts from `wrapped Ethereum WZCN token` to `native ZCN token`

### List of commands

| Contract | Address                                                         |
|----------|-----------------------------------------------------------------|
| bridge-auth-config   | Show authorizer configurations                                  |
| bridge-auth-config-update | Update ZCNSC authorizer settings by ID                          |
| bridge-burn-eth | Burn eth tokens                                                 |
| bridge-burn-zcn | Burn zcn tokens                                                 |
| bridge-client-init | Init bridge client config (bridge.yaml) in HOME (~/.zcn) folder |
| bridge-config | Show ZCNBridge configurations                                   |
| bridge-config-update | Update ZCNSC bridge global settings                             |
| bridge-get-wzcn-burn | Get confirmed burn ticket for ethereum burn transaction         |
| bridge-get-zcn-burn | Get the confirmed burn ticket for zcn burn transaction          |
| bridge-import-account | Import Ethereum account to local key storage (default $HOME/.zcn/wallets) |
| bridge-list-accounts | List Ethereum account registered in local key chain             |
| bridge-list-auth | List authorizers                                                |
| bridge-mint-wzcn | Mint WZCN tokens using the hash of ZCN burn transaction         |
| bridge-mint-zcn | Mint zcn tokens using the hash of Ethereum burn transaction     |
| bridge-owner-init | Init bridge owner config (owner.yaml) in HOME (~/.zcn) folder   |
| bridge-verify | Verify ethereum transaction                                     |

##### bridge-client-init

```text
Flags:
--bridge_config string       Bridge config file name (default "bridge.yaml")
--bridgeaddress string       Bridge contract address
--consensusthreshold float   Consensus threshold required to reach consensus for burn tickets (default 0.75)
--ethereumaddress string     Client Ethereum address
--ethereumnodeurl string     Ethereum Node URL (Infura/Alchemy)
--gaslimit int               appr. Gas limit to execute Ethereum transaction (default 300000)
--password string            Password to be used to unlock private key stored in local storage
--path string                Configuration directory (default "/Users/dima/.zcn")
--value int                  Value sent along with Ethereum transaction
--wzcnaddress string         WZCN token address

Global Flags:
--config string      config file (default is config.yaml)
--configDir string   configuration directory (default is $HOME/.zcn)
--network string     network file to overwrite the network details (if required, default is network.yaml)
--silent             Do not print sdk logs in stderr (prints by default)
--wallet string      wallet file (default is wallet.json)
```

##### bridge-import-account

```text
Flags:
  -h, --help              help for bridge-import-account
      --mnemonic string   Ethereum mnemonic
      --password string   Password to lock and unlock account to sign transaction
      --path string       Home config directory (default "/Users/dima/.zcn")

Global Flags:
      --config string      config file (default is config.yaml)
      --configDir string   configuration directory (default is $HOME/.zcn)
      --network string     network file to overwrite the network details (if required, default is network.yaml)
      --silent             Do not print sdk logs in stderr (prints by default)
      --wallet string      wallet file (default is wallet.json)
```

# Demo

## Convert from ZCN to WZCN token using separate steps

1. Create ethereum wallet account
```shell
./zwallet bridge-import-account \
            --mnemonic "coconut moral level thing soldier excess ranch swim balance magnet embrace beef" \
            --password "password"
```
2. List ethereum accounts in key vault

```shell
./zwallet bridge-list-accounts
```

```text
Flags:
  -h, --help          help for bridge-list-accounts
      --path string   Configuration dir (default "/Users/dima/.zcn")

Global Flags:
      --config string      config file (default is config.yaml)
      --configDir string   configuration directory (default is $HOME/.zcn)
      --network string     network file to overwrite the network details (if required, default is network.yaml)
      --silent             Do not print sdk logs in stderr (prints by default)
      --wallet string      wallet file (default is wallet.json)
```


3. Fill ethereum wallet with tokens  
Must be performed with metamask

4. Create ZCN wallet and fill it with tokens
```shell
for run in {1..10}
do
    ./zwallet faucet --methodName pour --input “{Pay day}”
done
```
5. Burn ZCN tokens

Using this command burn some tokens in ZCN

```shell
./zwallet bridge-burn-zcn --token 1 
```

5. Mint WZCN tokens

Using this command mint some tokens using hash from previous step

```shell
./zwallet bridge-mint-eth --hash "zcn-burn-hash" 
```

## Convert from WZCN to ZCN token using separate steps

1. Create ethereum wallet account
```shell
./zwallet bridge-import-account \
            --mnemonic "coconut moral level thing soldier excess ranch swim balance magnet embrace beef" \
            --password "password"
```
2. Fill ethereum wallet with tokens
3. Create ZCN wallet and fill it with tokens

```shell
for run in {1..10}
do
    ./zwallet faucet --methodName pour --input “{Pay day}”
done
```

4. Burn WZCN tokens

Using this command burn some tokens in Ethereum

```shell
./zwallet bridge-burn-eth --amount 1 #wei 
```

5. Mint ZCN tokens

Using this command mint tokens in ZCN using hash from previous step

```shell
./zwallet bridge-mint-zcn --hash "eth-burn-hash" 
```

## Convert from ZCN to WZCN with one step
## Convert from WZCN to ZCN with one step
## Ethereum's transaction verification

```shell
./zwallet bridge-verify --hash "eth-transaction-hash"
```

## List of authorizers
```shell
./zwallet bridge-list-auth
```

## Add authorizer

## Schema

![Diagram](https://github.com/Dmdv/0chain/blob/zcnsc_latest/code/go/0chain.net/smartcontract/zcnsc/docs/ZCNSC.png?raw=true)

## Converting WZCN to ZCN

1. The client owning WZCN writes a transaction to the WZCN mint to burn  
   tokens; we refer to this transaction as the WZCN burn transaction. This  
   transaction contains:  
   • The amount of WZCN burned.  
   • The client’s ID on the 0chain blockchain.  
   • A sequential nonce. (This nonce is distinct from the nonce used in Ethereum’s protocol.)  
2. The Ethereum network accepts the transaction and includes it in the blockchain.  
   Note that it will only be accepted if the nonce is one greater than the previous nonce.  
   The authorizers monitor the Ethereum blockchain for WZCN burn transactions.  
3. Each authorizer verifies that the transaction has been accepted on the  
   Ethereum blockchain. If the request is valid, the authorizer sends the  
   client a proof-of-WZCN-burn ticket. This ticket contains:  
   • The Ethereum transaction ID.  
   • The amount of ZCN to be minted.  
   • The client ID to receive the ZCN.  
   • The nonce provided the client in step 1 above.  
   • The authorizer’s signature.  
4. Once the client has gathered sufficient tickets, they write a ZCN mint  
   transaction containing these tickets. The 0chain miners verify the validity  
   of the tickets, checking the signatures and the nonces. If the ZCN mint  
   transaction is valid, the miners accept the transaction and allocate new  
   ZCN to the client’s account.  

## Converting ZCN to WZCN

The reverse process is similar. A client burns ZCN and presents proof to the  
authorizers, who then authorize the generation of new WZCN tokens.  
The steps for this process are as follows:  

1. The client writes a ZCN burn transaction, destroying ZCN tokens. This  
   transaction includes:  
   • The amount of ZCN to burn.  
   • The Ethereum address to receive the new WZCN.  
   • The nonce value.  
2. The 0Chain miners accept this transaction if the nonce is valid and the  
   client has sufficient ZCN  
3. Each authorizer monitors the 0Chain blockchain for a ZCN burn transaction.  
   Once the transaction is accepted, the authorizer sends the client a  
   proof-of-ZCN-burn ticket, containing similar information as the proof-of-  
   WZCN-burn ticket:  
   • The 0Chain transaction ID.  
   • The nonce value.  
   • The amount of WZCN to be minted.  
   • The Ethereum address to receive the WZCN.  
   • The authorizer’s signature.  
4. Once the client has gathered sufficient tickets, they write a transaction to  
   the WZCN mint, including the tickets from the authorizers.  
5. The WZCN mint verifies the validity of the tickets, checking the signatures  
   and the nonce values. If the transaction is valid, the WZCN mint creates  
   new WZCN for the client.  
