###############################################
# Requires `kar-actor-example` to be running
###############################################
#!/bin/sh

CLASSPATH=../target/kar-rest-client.jar:../target/libs/*

 kar -runtime_port 32123 -app example java -cp $CLASSPATH test.RunService