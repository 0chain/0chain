/*
 * Copyright 2012-2016 the original author or authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package net.zerochain.Block;

import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Service;
import net.zerochain.resources.Utils;
import net.zerochain.resources.crypto.asymmetric.*;
import net.zerochain.Transaction.ITransactionService;
import net.zerochain.BlockDAO.IBlockDAO;
import net.zerochain.Transaction.TransactionEntity;
import net.zerochain.Client.ClientEntity;
import java.util.List;
import java.util.ArrayList;
import java.security.KeyPair;
import net.zerochain.Block.BlockEntity;
import net.zerochain.Block.BlockTransactionEntity;

import org.springframework.web.client.RestTemplate;
import org.springframework.http.ResponseEntity;

@Service("blockService")
public class BlockServiceImpl implements IBlockService {

	@Autowired 
	private ITransactionService iTransactionService;

	@Autowired
	private IBlockDAO iBlockDAO;

	private ClientEntity minerEntity;
	private String private_key;
	private AsymmetricSigning algo;

	@Override
	public BlockEntity generateBlock() {
		List<TransactionEntity> transactions = iTransactionService.getTwoHundredTransactions();
		List<String> hash_msg = iTransactionService.verifyTransactionsWithoutTime(transactions);
		iTransactionService.updateTransactionsToPending(transactions);
		
		BlockEntity blockEntity = createBlockEntity(transactions);
		
		iBlockDAO.saveBlock(blockEntity);
		saveTransactions(blockEntity,hash_msg);
		return blockEntity;
	}

	@Override
	public void sendBlock(BlockEntity blockEntity)
	{

	    RestTemplate restTemplate = new RestTemplate();
	    ResponseEntity<String> responseEntity = restTemplate.postForEntity("http://localhost:8081/v1/block", blockEntity, String.class);
		String response = responseEntity.getBody();
	}

	@Override
	public void saveTransactions(BlockEntity blockEntity, List<String> hash_msg)
	{
		for(String hash : hash_msg)
		{
			iBlockDAO.saveBlockTransaction(new BlockTransactionEntity(blockEntity.getBlock_hash(),hash));
		}
	}

	@Override
	public BlockEntity createBlockEntity(List<TransactionEntity> transactions)
	{
		BlockEntity blockEntity = new BlockEntity();
		String prev_block_hash = "";
		int round = 0;

		if(!iBlockDAO.isBlockEmpty())
		{
			prev_block_hash = iBlockDAO.getLastFinalizedBlockHash();
			round = iBlockDAO.getLastFinalizedRound()+1;
		}
		blockEntity.setBlock_hash(createBlockHash(transactions,prev_block_hash));
		blockEntity.setPrev_block_hash(prev_block_hash);
		blockEntity.setBlock_signature(algo.createSignature(private_key,blockEntity.getBlock_hash()));
		blockEntity.setMiner_id(minerEntity.getClientID());
		blockEntity.setRound(round);
		blockEntity.setTimestamp(Utils.getTimestamp());
		return blockEntity;
	}

	@Override
	public String createBlockHash(List<TransactionEntity> transactions, String prev_block_hash)
	{
		String concat = "";
		String blockHash = "";
		for(TransactionEntity t:transactions)
		{
			concat = concat + t.getHash_msg();
		}
		blockHash = Utils.createHash(concat+prev_block_hash);
		return blockHash;
	}

	@Override
	public void setMiner()
	{
		minerEntity = new ClientEntity();
		algo = new EDDSA();
		KeyPair keys = algo.createKeys();
		String public_key = Utils.toHexString(keys.getPublic().getEncoded());
		private_key = Utils.toHexString(keys.getPrivate().getEncoded());
		String hash_key = Utils.createHash(public_key);
		minerEntity.setPublic_key(public_key);
		minerEntity.setClientID(hash_key);
	}

}
