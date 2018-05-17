package net.zerochain.component;

import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Component;
import org.springframework.boot.CommandLineRunner;
import net.zerochain.Client.IClientService;

@Component
public class ClientComponent implements CommandLineRunner
{
	@Autowired
	IClientService iClientService;

	@Override
	public void run(String... args) throws Exception
	{
		iClientService.setClient();
		iClientService.sendClient();
		iClientService.sendTransactions(1000000000L);
		System.exit(0);
	}
}