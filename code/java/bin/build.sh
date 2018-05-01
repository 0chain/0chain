#!/bin/bash

modules[0]=client
modules[1]=transaction
modules[2]=block

resources[0]=utils


main()
{
	if [ ! -f swagger-codegen-cli.jar ]; then
		curl http://central.maven.org/maven2/io/swagger/swagger-codegen-cli/2.3.1/swagger-codegen-cli-2.3.1.jar -o swagger-codegen-cli.jar
	fi
	createAndGoToBuildDirectory
	createUtilDirectory
	createImplementationDirectory

	generateSwaggerServers ${modules[0]}
	cd utils
	/opt/apache-maven-3.5.3/bin/mvn install
	cd ../api/clientServer
	/opt/apache-maven-3.5.3/bin/mvn install
	cd ../../impl
	/opt/apache-maven-3.5.3/bin/mvn install
	cd ../
	addClientServerFiles
	cd ./api/clientServer
	/opt/apache-maven-3.5.3/bin/mvn package


}

createAndGoToBuildDirectory()
{
	mkdir ../build
	cd ../build
}

createUtilDirectory()
{
	mkdir -p utils/src/main/java/net/chain0/resources/
	mkdir -p utils/src/test/java/net/chain0/resources/crypto/asymmetric
	cp ../bin/poms/resources/pom.xml ./utils/pom.xml
	cp -R ../src/. ./utils/src/main/java/net/chain0/resources/
}

createImplementationDirectory()
{
	mkdir -p impl/src/main/java/net/chain0/
	cp ../bin/poms/impl/pom.xml ./impl/pom.xml
	cp -R ../swagger-impl/. ./impl/src/main/java/net/chain0
}

generateSwaggerServers()
{
	java -jar ../bin/swagger-codegen-cli.jar generate \
		-i ../../swagger/$1.yaml \
		-l jaxrs \
		-o api/$1Server \
		-c ../bin/config/config-server-$1.json

	java -jar ../bin/swagger-codegen-cli.jar generate \
		-i ../../swagger/$1.yaml \
		-l java \
		-o api/$1Client \
		-c ../bin/config/config-client-$1.json
}


addClientServerFiles()
{
	cp ../bin/poms/clientServer/pom.xml ./api/clientServer/pom.xml
	cp ../bin/serviceImpl/clientServer/ClientApiServiceImpl.java ./api/clientServer/src/main/java/net/chain0/client/api/impl/ClientApiServiceImpl.java
}

main
