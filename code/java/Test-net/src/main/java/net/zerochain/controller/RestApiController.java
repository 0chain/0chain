package net.zerochain.controller;

import org.apache.log4j.Logger;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestMethod;
import org.springframework.web.bind.annotation.RestController;

import net.zerochain.Client.IClientService;
import net.zerochain.Client.ClientEntity;
import net.zerochain.Transaction.ITransactionService;
import net.zerochain.Transaction.TransactionEntity;
import net.zerochain.Response.Response;

@RestController
@RequestMapping("/v1")
public class RestApiController {
	private static Logger logger = Logger.getLogger(RestApiController.class);

	@Autowired 
	IClientService iClientService;
	
	@Autowired 
	ITransactionService iTransactionService;
	
	
	//--------- Post Registration--------
	
	@RequestMapping(value = "/client", method = RequestMethod.POST)
	public ResponseEntity<Response> postRegistration(@RequestBody ClientEntity clientEntity){
		logger.info("Recieved Client: "+ clientEntity);
		Response test = iClientService.verifyNewClient(clientEntity);
		if(test.getName().equals("Success"))
		{
			logger.info("Adding Client: "+ clientEntity);
			iClientService.saveRegistration(clientEntity);
			return new ResponseEntity<Response> (test, HttpStatus.OK);
		}
		else
		{
			logger.info("Client failed: "+clientEntity+"\n"+test);
			return new ResponseEntity<Response> (test, HttpStatus.BAD_REQUEST);
		}
	}
	
	//---------- Post Transaction --------
	@RequestMapping(value ="/transaction", method = RequestMethod.POST)
	public ResponseEntity<Response> postTransaction(@RequestBody TransactionEntity transactionEntity){
		//System.out.println("In Transaction");
		logger.info("Recieved Transaction: "+ transactionEntity);
		Response test = iTransactionService.verifyNewTransaction(transactionEntity);
		if(test.getName().equals("Success"))
		{
			logger.info("Adding Transaction: "+transactionEntity);
			iTransactionService.saveTransaction(transactionEntity);
			return new ResponseEntity<Response> (test, HttpStatus.OK);
		}
		else
		{
			logger.info("Transaction failed: "+transactionEntity+'\n'+test);
			return new ResponseEntity<Response> (test, HttpStatus.BAD_REQUEST);
		}
	}

}
