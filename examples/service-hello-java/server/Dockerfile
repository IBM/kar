ARG JAVA_BUILDER=us.icr.io/research/kar-dev/sdk-java-builder-11
ARG JAVA_RUNTIME=us.icr.io/research/kar-dev/sdk-java-runtime-11

#
# Build the artifacts using the builder image
#
FROM $JAVA_BUILDER as builder

WORKDIR /kar/app
COPY pom.xml ./
COPY src src
RUN mvn package

#
# Copy the artifacts to the runtime image
#
FROM $JAVA_RUNTIME
COPY --from=builder --chown=default:root /kar/app/target/kar-hello-server.war /opt/ol/wlp/usr/servers/defaultServer/apps
COPY --from=builder --chown=default:root /kar/app/src/main/liberty/config/server.xml /opt/ol/wlp/usr/servers/defaultServer

#
# Run Open Liberty configure script.
# Doing this now reduces startup time of the final image
#
RUN /opt/ol/helpers/build/configure.sh default
