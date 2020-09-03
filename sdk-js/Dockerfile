ARG KAR_BASE=us.icr.io/research/kar-dev/kar

FROM $KAR_BASE as kar

FROM node:12-alpine

# S2I setup
COPY s2i/bin /s2i
LABEL io.openshift.s2i.scripts-url=image:///s2i

# Copy in kar cli to enable "local mode" style when running under IBM Code Engine
COPY --from=kar /kar/bin/kar /kar/bin/kar
