#!/bin/bash

BLOBBERID=7a90e6790bcd3d78422d7a230390edc102870fe58c15472073922024985b1c7d
echo
echo blobber: $BLOBBERID
echo -e "\naverges"
echo -e "\ngraph-blobber-write-price"
curl --location -g --request GET  'http://192.168.1.100:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/graph-blobber-write-price?data-points=17&id='$BLOBBERID
echo -e "\ngraph-blobber-capacity"
curl --location -g --request GET  'http://192.168.1.100:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/graph-blobber-capacity?data-points=17&id='$BLOBBERID
echo -e "\ngraph-blobber-allocated"
curl --location -g --request GET  'http://192.168.1.100:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/graph-blobber-allocated?data-points=17&id='$BLOBBERID
echo -e "\ngraph-blobber-saved-data"
curl --location -g --request GET  'http://192.168.1.100:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/graph-blobber-saved-data?data-points=17&id='$BLOBBERID
echo -e "\ngraph-blobber-read-data"
curl --location -g --request GET  'http://192.168.1.100:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/graph-blobber-read-data?data-points=17&id='$BLOBBERID
echo -e "\ngraph-blobber-offers-total"
curl --location -g --request GET  'http://192.168.1.100:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/graph-blobber-offers-total?data-points=17&id='$BLOBBERID
echo -e "\ngraph-blobber-unstake-total"
curl --location -g --request GET  'http://192.168.1.100:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/graph-blobber-unstake-total?data-points=17&id='$BLOBBERID
echo -e "\ngraph-blobber-total-stake"
curl --location -g --request GET  'http://192.168.1.100:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/graph-blobber-total-stake?data-points=17&id='$BLOBBERID
echo -e "\ngraph-blobber-challenges-open"
curl --location -g --request GET  'http://192.168.1.100:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/graph-blobber-challenges-open?data-points=17&id='$BLOBBERID

echo -e "\ndifferences"
echo -e "\ngraph-blobber-service-charge"
curl --location -g --request GET  'http://192.168.1.100:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/graph-blobber-service-charge?data-points=17&id='$BLOBBERID
echo -e "\ngraph-blobber-challenges-passed"
curl --location -g --request GET  'http://192.168.1.100:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/graph-blobber-challenges-passed?data-points=17&id='$BLOBBERID
echo -e "\ngraph-blobber-challenges-completed"
curl --location -g --request GET  'http://192.168.1.100:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/graph-blobber-challenges-completed?data-points=17&id='$BLOBBERID
echo -e "\ngraph-blobber-inactive-rounds"
curl --location -g --request GET  'http://192.168.1.100:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/graph-blobber-inactive-rounds?data-points=17&id='$BLOBBERID

echo -e  "\ngobal endpoints"
echo -e "\nsingle point"
echo -e "\ntotal-minted"
curl --location -g --request GET  'http://192.168.1.100:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/total-minted'
echo -e "\ntotal-blobber-capacity"
curl --location -g --request GET  'http://192.168.1.100:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/total-blobber-capacity'
echo -e "\naverage-write-price"
curl --location -g --request GET  'http://192.168.1.100:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/average-write-price'
echo -e "\ntotal-successful-challenges"
curl --location -g --request GET  'http://192.168.1.100:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/total-successful-challenges'
echo -e "\ntotal-total-challenges"
curl --location -g --request GET  'http://192.168.1.100:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/total-total-challenges'
echo -e "\ntotal-allocated-storage"
curl --location -g --request GET  'http://192.168.1.100:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/total-allocated-storage'

echo -e "\ngraph point"
echo -e "\ngraph-average-write-price"
curl --location -g --request GET  'http://192.168.1.100:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/graph-average-write-price?data-points=17'
echo -e "\ngraph-total-challenge-pools"
curl --location -g --request GET  'http://192.168.1.100:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/graph-total-challenge-pools?data-points=17'

echo -e "\ngraph-allocated-storage"
curl --location -g --request GET  'http://192.168.1.100:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/graph-allocated-storage?data-points=17'
echo -e "\ngraph-used-storage"
curl --location -g --request GET  'http://192.168.1.100:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/graph-used-storage?data-points=17'
echo -e "\ngraph-total-locked"
curl --location -g --request GET  'http://192.168.1.100:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/graph-total-locked?data-points=17'
echo -e "\ngraph-total-minted"
curl --location -g --request GET  'http://192.168.1.100:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/graph-total-minted?data-points=17'

echo -e "\ngraph-total-staked"
curl --location -g --request GET  'http://192.168.1.100:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/graph-total-staked?data-points=17'
echo -e "\ngraph-challenges"
curl --location -g --request GET  'http://192.168.1.100:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/graph-challenges?data-points=17'
echo -e "\ngraph-token-supply"
curl --location -g --request GET  'http://192.168.1.100:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/graph-token-supply?data-points=17'
