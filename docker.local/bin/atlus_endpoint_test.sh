#!/bin/bash

BLOBBERID=2f051ca6447d8712a020213672bece683dbd0d23a81fdf93ff273043a0764d18
echo
echo blobber $BLOBBERID
echo -e "\naverges"
echo -e "\ngraph-blobber-average-write-price"
curl --location -g --request GET  'http://192.168.1.100:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/graph-blobber-write-price?data-points=17&id='$BLOBBERID
echo -e "\ngraph-blobber-average-capacity"
curl --location -g --request GET  'http://192.168.1.100:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/graph-blobber-capacity?data-points=17&id='$BLOBBERID
echo -e "\ngraph-blobber-average-allocated"
curl --location -g --request GET  'http://192.168.1.100:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/graph-blobber-allocated?data-points=17&id='$BLOBBERID
echo -e "\ngraph-blobber-average-saved-data"
curl --location -g --request GET  'http://192.168.1.100:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/graph-blobber-saved-data?data-points=17&id='$BLOBBERID
echo -e "\ngraph-blobber-average-offers-total"
curl --location -g --request GET  'http://192.168.1.100:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/graph-blobber-offers-total?data-points=17&id='$BLOBBERID
echo -e "\ngraph-blobber-average-unstake-total"
curl --location -g --request GET  'http://192.168.1.100:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/graph-blobber-unstake-total?data-points=17&id='$BLOBBERID
echo -e "\ngraph-blobber-average-total-stake"
curl --location -g --request GET  'http://192.168.1.100:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/graph-blobber-total-stake?data-points=17&id='$BLOBBERID

ech -e "\ndifferences"
echo -e "\ngraph-blobber-service-charge"
curl --location -g --request GET  'http://192.168.1.100:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/graph-blobber-service-charge?data-points=17&id='$BLOBBERID
echo -e "\ngraph-blobber-challenges-passed"
curl --location -g --request GET  'http://192.168.1.100:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/graph-blobber-challenges-passed?data-points=17&id='$BLOBBERID
echo -e "\ngraph-blobber-challenges-completed"
curl --location -g --request GET  'http://192.168.1.100:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/graph-blobber-challenges-completed?data-points=17&id='$BLOBBERID
echo -e "\ngraph-blobber-inactive-rounds"
curl --location -g --request GET  'http://192.168.1.100:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/graph-blobber-inactive-rounds?data-points=17&id='$BLOBBERID

echo -e  "\ngobal endpoints"
echo -e "\nsingale point"
echo -e "\ntotal-mint"
curl --location -g --request GET  'http://192.168.1.100:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/total-mint'
echo -e "\ntotal-blobber-capacity"
curl --location -g --request GET  'http://192.168.1.100:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/total-blobber-capacity'
echo -e "\naverage-write-price"
curl --location -g --request GET  'http://192.168.1.100:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/average-write-price'
echo -e "\ntotal-stored-data"
curl --location -g --request GET  'http://192.168.1.100:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/total-stored-data'
echo -e "\ntotal-successful-challenges"
curl --location -g --request GET  'http://192.168.1.100:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/total-successful-challenges'
echo -e "\ntotal-total-challenges"
curl --location -g --request GET  'http://192.168.1.100:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/total-total-challenges'
echo -e "\ntotal-allocated storage"
curl --location -g --request GET  'http://192.168.1.100:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/total-allocated storage'

echo -e "\ngraph point"
echo -e "\ndgraph-average-write-price"
curl --location -g --request GET  'http://192.168.1.100:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/graph-average-write-price?data-points=17'
echo -e "\ngraph-data-storage-cost"
curl --location -g --request GET  'http://192.168.1.100:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/graph-data-storage-cost?data-points=17'

echo -e "\ngraph-allocated-storage"
curl --location -g --request GET  'http://192.168.1.100:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/graph-allocated-storage?data-points=17'
echo -e "\ngraph-used-storage"
curl --location -g --request GET  'http://192.168.1.100:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/graph-used-storage?data-points=17'
echo -e "\ngraph-total-locked"
curl --location -g --request GET  'http://192.168.1.100:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/graph-total-locked?data-points=17'
echo -e "\ngraph-total-minted"
curl --location -g --request GET  'http://192.168.1.100:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/graph-total-minted?data-points=17'

echo -e "\ngraph-cloud-growth"
curl --location -g --request GET  'http://192.168.1.100:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/graph-cloud-growth?data-points=17'
echo -e "\ngraph-total-staked"
curl --location -g --request GET  'http://192.168.1.100:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/graph-total-staked?data-points=17'
echo -e "\ngraph-network-quality"
curl --location -g --request GET  'http://192.168.1.100:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/graph-network-quality?data-points=17'
echo -e "\ngraph-zcn-supply"
curl --location -g --request GET  'http://192.168.1.100:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/graph-zcn-supply?data-points=17'
