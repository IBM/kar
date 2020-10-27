FROM adoptopenjdk/openjdk11:alpine

WORKDIR /workspace

COPY dependencies dependencies/
COPY src src/
COPY properties properties/

ENV CAMEL_K_CONF_D=properties
CMD assemble() { printf "file:"; while [ $# -gt 1 ]; do printf "%s%s" "$1" ",file:"; shift; done; printf "%s" "$1"; } ; CAMEL_K_ROUTES=$(assemble $(find src -type f)) java -cp "dependencies/*" org.apache.camel.k.main.Application
