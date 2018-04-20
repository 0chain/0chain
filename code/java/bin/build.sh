#!/bin/bash

buildDirectoryPath="../build"

main ()
{
	case $1 in
		build)
			case $2 in
				all)
					createBuildDirectory
					swaggerGenerate
					updateSwagerFilesAfterBuild
					generateIntegrationTests
					generateUtils
					generateSources
					installDependencies
					packageServers
					;;
				registration)
					if [ -d $buildDirectoryPath ]; then
						goToBuildDirectory
						swaggerGenerateRegistration
						updateRegistrationFilesAfterBuild
						generateRegistrationIntegrationTest
						installRegistrationClient
						pacakageRegistrationServer
					else
						echo "First, build all"
					fi
					;;
				transaction)
					if [ -d $buildDirectoryPath ]; then
						goToBuildDirectory
						swaggerGenerateTransaction
						updateTransactionFilesAferBuild
						generateTransactionIntegrationTest
						installTransactionClient
						packageTransactionServer
					else
						echo "First, build all"
					fi
					;;
				utils)
					if [ -d $buildDirectoryPath ]; then
						goToBuildDirectory
						generateUtils
						installUtils
					else
						echo "First, build all"
					fi
					;;
				integrationTest)
					if [ -d $buildDirectoryPath ]; then
						goToBuildDirectory
						generateIntegrationTests
						installTransactionClient
						installRegistrationClient
					else
						echo "First, build all"
					fi
					;;
			esac
			;;
		update)
			case $2 in
				all)
				if [ -d $buildDirectoryPath ]; then
					goToBuildDirectory
					installDependencies
					packageServers
				fi
				;;
				registration)
				if [ -d $buildDirectoryPath ]; then
					goToBuildDirectory
					generateSources
					installRegistrationClient
					pacakageRegistrationServer
				fi
				;;
				transaction)
				if [ -d $buildDirectoryPath ]; then
					goToBuildDirectory
					generateSources
					installTransactionClient
					packageTransactionServer
				fi
				;;
			esac
			;;
		test)
			case $2 in
				all)
				if [ -d $buildDirectoryPath"/transactionTest" ] && [ -d $buildDirectoryPath"/registrationTest" ]; then
					goToBuildDirectory
					integrationTestBoth
				fi
				;;
				transaction)
				if [ -d $buildDirectoryPath"/transactionTest" ]; then
					goToBuildDirectory
					integrationTestTransaction
				fi
				;;
				registration)
				if [ -d $buildDirectoryPath"/registrationTest" ]; then
					goToBuildDirectory
					integrationTestRegistration
				fi
				;;
			esac
	esac
				
}

createBuildDirectory()
{
	cd .. 

	mkdir build

	cd build

	cp ../bin/poms/pom.xml ./pom.xml
}

goToBuildDirectory()
{
	cd $buildDirectoryPath
}

swaggerGenerate()
{
	swaggerGenerateRegistration
	swaggerGenerateTransaction
}

swaggerGenerateRegistration()
{
	java -jar ../bin/swagger-codegen-cli.jar generate \
		-i ../../swagger/registration.yaml \
		-l jaxrs \
		-o registrationServer \
		-c ../src/config-server-registration.json

	java -jar ../bin/swagger-codegen-cli.jar generate \
		-i ../../swagger/registration.yaml \
		-l java \
		-o registrationClient \
		-c ../src/config-client-registration.json
}

swaggerGenerateTransaction()
{
	java -jar ../bin/swagger-codegen-cli.jar generate \
		-i ../../swagger/transaction.yaml \
		-l java \
		-o transactionClient \
		-c ../src/config-client-transaction.json

	java -jar ../bin/swagger-codegen-cli.jar generate \
		-i ../../swagger/transaction.yaml \
		-l jaxrs \
		-o transactionServer \
		-c ../src/config-server-transaction.json
}

updateSwagerFilesAfterBuild()
{
	updateTransactionFilesAferBuild
	updateRegistrationFilesAfterBuild	
}

updateTransactionFilesAferBuild()
{
	cp ../bin/poms/transactionServer/pom.xml transactionServer

	sed -i -e 's|<url-pattern>/transaction/|<url-pattern>\/|' transactionServer/src/main/webapp/WEB-INF/web.xml

	rm transactionServer/src/main/webapp/WEB-INF/web.xml-e
}

updateRegistrationFilesAfterBuild()
{
	sed -i -e 's|String publicKey = null| String publicKey = ""|' registrationClient/src/main/java/net/chain0/client/registration/model/Client.java

	rm registrationClient/src/main/java/net/chain0/client/registration/model/Client.java-e

	sed -i -e 's|String clientID = null| String clientID = ""|' registrationClient/src/main/java/net/chain0/client/registration/model/Client.java

	rm registrationClient/src/main/java/net/chain0/client/registration/model/Client.java-e

	sed -i -e 's|String signature = null| String signature = ""|' registrationClient/src/main/java/net/chain0/client/registration/model/Client.java

	rm registrationClient/src/main/java/net/chain0/client/registration/model/Client.java-e

	cp ../bin/poms/registrationServer/pom.xml registrationServer

	sed -i -e 's|<url-pattern>/registration/|<url-pattern>\/|' registrationServer/src/main/webapp/WEB-INF/web.xml

	rm registrationServer/src/main/webapp/WEB-INF/web.xml-e
}

installDependencies()
{
	generateSources

	installUtils

	installRegistrationClient

	installTransactionClient

}

generateSources()
{
	/opt/apache-maven-3.5.3/bin/mvn generate-sources
}

installUtils()
{
	/opt/apache-maven-3.5.3/bin/mvn -pl utils clean install
}

installRegistrationClient()
{
	/opt/apache-maven-3.5.3/bin/mvn -pl registrationClient clean install
}

installTransactionClient()
{
	/opt/apache-maven-3.5.3/bin/mvn -pl transactionClient clean install
}

generateUtils()
{

	mkdir -p utils/src/main/java/net/chain0/resources

	mkdir -p utils/src/test/java/net/chain0/resources

	cp ../bin/poms/utils/pom.xml utils/pom.xml

}

generateIntegrationTests()
{
	generateRegistrationIntegrationTest
	generateTransactionIntegrationTest
}

generateRegistrationIntegrationTest()
{
	mkdir -p registrationTest/src/test/java/net/chain0/integrationTest

	cp ../bin/poms/registrationIntegrationTest/pom.xml registrationTest/pom.xml

	cp ../src/test/net/chain0/integrationTest/acceptClientIntegrationTest.java registrationTest/src/test/java/net/chain0/integrationTest/acceptClientIntegrationTest.java
}

generateTransactionIntegrationTest()
{
	mkdir -p transactionTest/src/test/java/net/chain0/integrationTest

	cp ../bin/poms/transactionIntegrationTest/pom.xml transactionTest/pom.xml

	cp ../src/test/net/chain0/integrationTest/acceptTransactionIntegrationTest.java transactionTest/src/test/java/net/chain0/integrationTest/acceptTransactionIntegrationTest.java
}

packageServers()
{
	packageTransactionServer
	pacakageRegistrationServer
}

packageTransactionServer()
{
	/opt/apache-maven-3.5.3/bin/mvn -pl transactionServer package
}

pacakageRegistrationServer()
{
	/opt/apache-maven-3.5.3/bin/mvn -pl registrationServer package
}

integrationTestBoth()
{
	/opt/apache-maven-3.5.3/bin/mvn -pl transactionTest,registrationTest test -Dtest=**/*IntegrationTest
}

integrationTestRegistration()
{
	/opt/apache-maven-3.5.3/bin/mvn -pl registrationTest test -Dtest=**/*IntegrationTest
}

integrationTestTransaction()
{
	/opt/apache-maven-3.5.3/bin/mvn -pl transactionTest test -Dtest=**/*IntegrationTest
}

main $1 $2