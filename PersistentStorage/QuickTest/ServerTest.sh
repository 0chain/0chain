#!/bin/bash

if [ -z "$1" ]
  then
    echo "No IP address argument supplied"
    echo "usage:  ./ServerTest.sh 172.33.1.134 8082"
    exit -1;
fi

if [ -z "$2" ]
  then
    echo "No port argument supplied"
    echo "usage:  ./ServerTest.sh 172.33.1.134 8082"
    exit -1;
fi

SERVER_IP=${1}
SERVER_PORT=${2}

####### CLIENT #######
###INSERT###
echo "Inserting client"
CLIENT_INSERT_RESPONSE=`curl -s -S -H "Content-Type: application/json" -X POST -d '{"public_key":"a9be6f99e77acc3345f726c22327d2d9ee38a5d66624a6553a4d3eb2949b3b5d","clientid":"6e54a7f91a44a5ab79859de8e495b335d1749665911c1aef94b725fafdd3f82f"}' http://${SERVER_IP}:${SERVER_PORT}/clients`;
if [ -z "${CLIENT_INSERT_RESPONSE}" ]
then
    echo "  Client insert connection failure!"
    echo "  Exiting"
    exit;
else
    CLIENT_INSERT_RESULT=`echo ${CLIENT_INSERT_RESPONSE} | cut -d":" -f3 | cut -d"\"" -f2`;
    CLIENT_INSERT_MESSAGE=`echo ${CLIENT_INSERT_RESPONSE} | cut -d":" -f4| cut -d"\"" -f2`;
    echo "  Client insert result is ${CLIENT_INSERT_RESULT} with message of \"${CLIENT_INSERT_MESSAGE}\"";
fi

###GET###
echo "Getting client"
CLIENT_GET_RESPONSE=`curl -s -S -H "Content-Type: application/json" -X GET -d '{"public_key":"","clientid":"6e54a7f91a44a5ab79859de8e495b335d1749665911c1aef94b725fafdd3f82f"}' http://${SERVER_IP}:${SERVER_PORT}/clients`;
if [ -z "${CLIENT_GET_RESPONSE}" ]
then
    echo "  Client get connection failure!"
else
    echo "  Get Response: ${CLIENT_GET_RESPONSE}"
fi


###PATCH###
echo "Updating client. Expect 404"
echo -n "  Patch Response:   "
curl -s -S -i -H "Content-Type: application/json" -X PATCH -d '{"public_key":"NEWVAL","clientid":"6e54a7f91a44a5ab79859de8e495b335d1749665911c1aef94b725fafdd3f82f"}' http://${SERVER_IP}:${SERVER_PORT}/clients | grep "HTTP/1.1"

###DEL###
echo "Delete client"
CLIENT_DELETE_RESPONSE=`curl -s -S -H "Content-Type: application/json" -X DELETE -d '{"public_key":"","clientid":"6e54a7f91a44a5ab79859de8e495b335d1749665911c1aef94b725fafdd3f82f"}' http://${SERVER_IP}:${SERVER_PORT}/clients`;
if [ -z "${CLIENT_DELETE_RESPONSE}" ]
then
    echo "  Client delete connection failure!"
else  
    CLIENT_DELETE_RESULT=`echo ${CLIENT_DELETE_RESPONSE} | cut -d":" -f3 | cut -d"\"" -f2`;
    CLIENT_DELETE_MESSAGE=`echo ${CLIENT_DELETE_RESPONSE} | cut -d":" -f4| cut -d"\"" -f2`;
    echo "  Client delete result is ${CLIENT_DELETE_RESULT} with message of \"${CLIENT_DELETE_MESSAGE}\"";
fi

###GET###
echo "Get deleted client. Expect Failure"
CLIENT_GET_RESPONSE=`curl -s -S -H "Content-Type: application/json" -X GET -d '{"public_key":"","clientid":"6e54a7f91a44a5ab79859de8e495b335d1749665911c1aef94b725fafdd3f82f"}' http://${SERVER_IP}:${SERVER_PORT}/clients`;
if [ -z "${CLIENT_GET_RESPONSE}" ]
then
    echo "  Client get connection failure!"
else
    CLIENT_GET_RESULT=`echo ${CLIENT_GET_RESPONSE} | cut -d":" -f3 | cut -d"\"" -f2`;
    CLIENT_GET_MESSAGE=`echo ${CLIENT_GET_RESPONSE} | cut -d":" -f4| cut -d"\"" -f2`;
    echo "  Client get result is ${CLIENT_GET_RESULT} with message of \"${CLIENT_GET_MESSAGE}\"";
fi
echo ""


####### TRANSACTION #######

###INSERT###
echo "Inserting transaction"
TRANSACTION_INSERT_RESPONSE=`curl -s -S -H "Content-Type: application/json" -X POST -d '{"clientid":"7c89ea8f7ca39246c5726aea3a4b8331a1b2715cf55403ccc415395de09fc961","transaction_data":"Sample transaction","createdate":"2018-05-10T18:45:43.928998+00:00","hash":"168b9977028f7509094a25c49d1d2ebd101389a97d2f18fd2cc679c84ee9b4a3","signature":"4427b84f5607d00b69365f8de89d72e98faee78e90ab52edf043881143fedc4dccf314ab5fcee5a25fbf59a443047e5c1289fcfd6aa25ce71ed3a4460021280f"}' http://${SERVER_IP}:${SERVER_PORT}/transactions`;
if [ -z "${TRANSACTION_INSERT_RESPONSE}" ]
then
    echo "  Transaction insert connection failure!"
    echo "  Exiting"
    exit;
else
    TRANSACTION_INSERT_RESULT=`echo ${TRANSACTION_INSERT_RESPONSE} | cut -d":" -f3 | cut -d"\"" -f2`;
    TRANSACTION_INSERT_MESSAGE=`echo ${TRANSACTION_INSERT_RESPONSE} | cut -d":" -f4| cut -d"\"" -f2`;
    echo "  Transaction insert result is ${TRANSACTION_INSERT_RESULT} with message of \"${TRANSACTION_INSERT_MESSAGE}\"";
fi

###GET###
echo "Getting transaction"
TRANSACTION_GET_RESPONSE=`curl -s -S -H "Content-Type: application/json" -X GET -d '{"clientid":"","transaction_data":"","createdate":"","hash":"168b9977028f7509094a25c49d1d2ebd101389a97d2f18fd2cc679c84ee9b4a3","signature":""}' http://${SERVER_IP}:${SERVER_PORT}/transactions`;
if [ -z "${TRANSACTION_GET_RESPONSE}" ]
then
    echo "  Transaction get connection failure!"
else    
    echo "  Get Response: ${TRANSACTION_GET_RESPONSE}"
fi

###PATCH###
echo "Updating transaction. Expect 404"
echo -n "  Patch Response:   "
curl -i -s -S -H "Content-Type: application/json" -X PATCH -d '{"clientid":"7c89ea8f7ca39246c5726aea3a4b8331a1b2715cf55403ccc415395de09fc961","transaction_data":"Sample transaction33","createdate":"2018-05-10T18:45:43.928998+00:00","hash":"168b9977028f7509094a25c49d1d2ebd101389a97d2f18fd2cc679c84ee9b4a3","signature":"4427b84f5607d00b69365f8de89d72e98faee78e90ab52edf043881143fedc4dccf314ab5fcee5a25fbf59a443047e5c1289fcfd6aa25ce71ed3a4460021280f"}' http://${SERVER_IP}:${SERVER_PORT}/transactions  | grep "HTTP/1.1";

###DEL###
echo "Delete transaction"
TRANSACTION_DELETE_RESPONSE=`curl -s -S -H "Content-Type: application/json" -X DELETE -d '{"clientid":"","transaction_data":"","createdate":"","hash":"168b9977028f7509094a25c49d1d2ebd101389a97d2f18fd2cc679c84ee9b4a3","signature":""}' http://${SERVER_IP}:${SERVER_PORT}/transactions`;
if [ -z "${TRANSACTION_DELETE_RESPONSE}" ]
then
    echo "  Transaction delete connection failure!"
else
    TRANSACTION_DELETE_RESULT=`echo ${TRANSACTION_DELETE_RESPONSE} | cut -d":" -f3 | cut -d"\"" -f2`;
    TRANSACTION_DELETE_MESSAGE=`echo ${TRANSACTION_DELETE_RESPONSE} | cut -d":" -f4| cut -d"\"" -f2`;
    echo "  Transaction delete result is ${TRANSACTION_DELETE_RESULT} with message of \"${TRANSACTION_DELETE_MESSAGE}\"";
fi

###GET###
echo "Get deleted transaction. Expect Failure."
TRANSACTION_GET_RESPONSE=`curl -s -S -H "Content-Type: application/json" -X GET -d '{"clientid":"7c89ea8f7ca39246c5726aea3a4b8331a1b2715cf55403ccc415395de09fc961","transaction_data":"","createdate":"","hash":"","signature":""}' http://${SERVER_IP}:${SERVER_PORT}/transactions`;
if [ -z "${TRANSACTION_DELETE_RESPONSE}" ]
then
    echo "  Transaction get connection failure!"
else
    TRANSACTION_GET_RESULT=`echo ${TRANSACTION_GET_RESPONSE} | cut -d":" -f3 | cut -d"\"" -f2`;
    TRANSACTION_GET_MESSAGE=`echo ${TRANSACTION_GET_RESPONSE} | cut -d":" -f4| cut -d"\"" -f2`;
    echo "  Transaction get result is ${TRANSACTION_GET_RESULT} with message of \"${TRANSACTION_GET_MESSAGE}\"";
fi
echo ""
exit;
