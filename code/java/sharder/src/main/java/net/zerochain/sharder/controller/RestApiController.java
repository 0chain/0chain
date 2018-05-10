package net.zerochain.sharder.controller;
import org.apache.log4j.Logger;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestMethod;
import org.springframework.web.bind.annotation.RestController;

import net.zerochain.sharder.Block.BlockEntity;
import net.zerochain.sharder.Block.IBlockService;

@RestController
@RequestMapping("/v1")
public class RestApiController {
	private static Logger logger = Logger.getLogger(RestApiController.class);
	
	@Autowired 
	IBlockService iBlockService;
	
	//-----PostBlock
	
	@RequestMapping (value = "/block", method = RequestMethod.POST)
	public ResponseEntity<?> postBlock(@RequestBody BlockEntity blockEntity){
		logger.info("Received Block" + blockEntity);
		iBlockService.saveBlock(blockEntity);
		return new ResponseEntity<String>("Success", HttpStatus.CREATED);
		
	}
}
