ARG KAR_BASE=us.icr.io/research/kar-dev/kar

FROM $KAR_BASE as kar
FROM adoptopenjdk/openjdk11:alpine

ENV LANG en_US.UTF-8
ENV LANGUAGE en_US:en
ENV LC_ALL en_US.UTF-8

RUN apk add --update maven && apk update && apk upgrade

# S2I setup
COPY ./s2i/bin/ /s2i
LABEL io.openshift.s2i.scripts-url=image:///s2i

# Copy in kar cli to enable "local mode" style when running under IBM Code Engine
RUN mkdir -p /kar/bin
COPY --from=kar /kar/kar /kar/bin/kar

WORKDIR /kar/sdk/java

COPY kar-java ./kar-java

WORKDIR /kar/sdk/java/kar-java

RUN mvn install
