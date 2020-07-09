FROM open-liberty

COPY --chown=1001:0 src/main/liberty/config /config/
COPY --chown=1001:0 target/kar-example-client.war /config/apps
RUN configure.sh