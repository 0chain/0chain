#!/bin/bash

BLOBBERID=2f051ca6447d8712a020213672bece683dbd0d23a81fdf93ff273043a0764d18
echo
echo blobber $BLOBBERID

echo -e "\nblobber-average-write-price"
curl --location -g --request GET  'http://192.168.1.100:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/blobber-average-write-price?data-points=17&id='$BLOBBERID
echo -e "\nblobber-average-capacity"
curl --location -g --request GET  'http://192.168.1.100:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/blobber-average-capacity?data-points=17&id='$BLOBBERID
echo -e "\nblobber-average-allocated"
curl --location -g --request GET  'http://192.168.1.100:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/blobber-average-allocated?data-points=17&id='$BLOBBERID
echo -e "\nblobber-average-saved-data"
curl --location -g --request GET  'http://192.168.1.100:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/blobber-average-saved-data?data-points=17&id='$BLOBBERID
echo -e "\nblobber-average-offers-total"
curl --location -g --request GET  'http://192.168.1.100:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/blobber-average-offers-total?data-points=17&id='$BLOBBERID
echo -e "\nblobber-average-unstake-total"
curl --location -g --request GET  'http://192.168.1.100:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/blobber-average-unstake-total?data-points=17&id='$BLOBBERID
echo -e "\nblobber-average-total-stake"
curl --location -g --request GET  'http://192.168.1.100:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/blobber-average-total-stake?data-points=17&id='$BLOBBERID


echo -e "\nblobber-service-charge"
curl --location -g --request GET  'http://192.168.1.100:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/blobber-service-charge?data-points=17&id='$BLOBBERID
echo -e "\nblobber-challenges-passed"
curl --location -g --request GET  'http://192.168.1.100:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/blobber-challenges-passed?data-points=17&id='$BLOBBERID
echo -e "\nblobber-challenges-completed"
curl --location -g --request GET  'http://192.168.1.100:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/blobber-challenges-completed?data-points=17&id='$BLOBBERID
echo -e "\nblobber-inactive-rounds"
curl --location -g --request GET  'http://192.168.1.100:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/blobber-inactive-rounds?data-points=17&id='$BLOBBERID

echo
echo gobal endpoints

echo -e "\ndata-storage-cost"
curl --location -g --request GET  'http://192.168.1.100:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/data-storage-cost?data-points=17'
echo -e "\ndaily-allocations"
curl --location -g --request GET  'http://192.168.1.100:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/daily-allocations?data-points=17'
echo -e "\ndaverage-rw-price"
curl --location -g --request GET  'http://192.168.1.100:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/average-rw-price?data-points=17'
echo -e "\ntotal-staked"
curl --location -g --request GET  'http://192.168.1.100:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/total-staked?data-points=17'
echo -e "\nnetwork-data-quality"
curl --location -g --request GET  'http://192.168.1.100:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/network-data-quality?data-points=17'
echo -e "\nzcn-supply"
curl --location -g --request GET  'http://192.168.1.100:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/zcn-supply?data-points=17'
echo -e "\nallocated-storage"
curl --location -g --request GET  'http://192.168.1.100:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/allocated-storage?data-points=17'
echo -e "\ncloud-growth"
curl --location -g --request GET  'http://192.168.1.100:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/cloud-growth?data-points=17'
echo -e "\ntotal-locked"
curl --location -g --request GET  'http://192.168.1.100:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/total-locked?data-points=17'
echo -e "\ndata-capitalization"
curl --location -g --request GET  'http://192.168.1.100:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/data-capitalization?data-points=17'
echo -e "\ndata-utilization"
curl --location -g --request GET  'http://192.168.1.100:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/data-utilization?data-points=17'



