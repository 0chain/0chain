package net.zerochain.sharder;

import org.apache.log4j.BasicConfigurator;
import org.apache.log4j.Logger;
import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.EnableAutoConfiguration;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.ComponentScan;
import org.springframework.context.support.PropertySourcesPlaceholderConfigurer;

@SpringBootApplication
@ComponentScan
@EnableAutoConfiguration
public class SharderApplication {
	private static Logger logger = Logger.getLogger(SharderApplication.class);

	public static void main(String[] args) {
		SpringApplication.run(SharderApplication.class, args);
		BasicConfigurator.configure();
		logger.info("Hello Sharder");
	}
	
	@Bean
    public static PropertySourcesPlaceholderConfigurer propertySourcesPlaceholderConfigurer() {
        return new PropertySourcesPlaceholderConfigurer();
    }
}
