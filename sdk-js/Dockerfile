ARG KAR_BASE=us.icr.io/research/kar-dev/kar

FROM $KAR_BASE as kar

FROM node:12-alpine

# Copy in kar cli to enable "local mode" style when running under IBM Code Engine
COPY --from=kar /kar/bin/kar /kar/bin/kar

# Inject wrapper script that detects local vs. sidecar mode
COPY runner /kar/bin/runner
CMD /kar/bin/runner
