ARG KAR_BASE=us.icr.io/research/kar-dev/kar

FROM $KAR_BASE as kar

FROM node:12-alpine

# Use non-root user "node" defined by node base image
RUN mkdir /kar && chown node /kar && chgrp node /kar && chmod g+s /kar
USER node

# Create exepcted /kar subdirs, will be owned by 'node'
RUN mkdir /kar/bin && mkdir /kar/app

# Copy in kar cli to enable "sidecar-in-container" mode
COPY --from=kar /kar/bin/kar /kar/bin/kar

# Inject wrapper script that detects local vs. sidecar mode
COPY runner /kar/bin/runner
CMD /kar/bin/runner
