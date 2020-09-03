FROM adoptopenjdk/openjdk11:alpine

ENV LANG en_US.UTF-8
ENV LANGUAGE en_US:en
ENV LC_ALL en_US.UTF-8

RUN apk add --update maven && apk update && apk upgrade

# S2I setup
COPY s2i/bin /s2i
LABEL io.openshift.s2i.scripts-url=image:///s2i

WORKDIR /kar/sdk-java

COPY pom.xml pom.xml
COPY kar-actor-runtime kar-actor-runtime
COPY kar-rest-client kar-rest-client

RUN mvn install
