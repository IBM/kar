ARG SDK_BASE=us.icr.io/research/kar-dev/sdk-nodejs-v12

FROM $SDK_BASE

WORKDIR /kar/examples/actors-ykt

COPY *.js *.json ./

RUN npm install --prod

ENV MAIN ykt.js

CMD node "$MAIN"
