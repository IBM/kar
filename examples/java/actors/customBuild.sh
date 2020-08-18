#!/bin/sh

mvn install

cd kar-actor-example
mvn liberty:package

