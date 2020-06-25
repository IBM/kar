###############################################
# Requires `kar-actor-example` to be running
###############################################
#!/bin/sh

CLASSPATH=../target/kar-rest-client.jar:../target/libs/*

 kar -runtime_port 32123 -app actor java -cp $CLASSPATH test.RunActor