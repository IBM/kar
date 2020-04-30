ARG SDK_BASE=us.icr.io/research/kar-dev/sdk-nodejs-v12

FROM $SDK_BASE

WORKDIR /kar/examples/incr

COPY *.js *.json ./

RUN npm install --prod

ENV MAIN server.js

CMD node "$MAIN"
