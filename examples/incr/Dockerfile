FROM node:12-alpine

ENV MAIN server.js

WORKDIR /kar/sdk/js

COPY sdk/js/*.js sdk/js/*.json ./

WORKDIR /kar/examples/incr

COPY examples/incr/*.js examples/incr/*.json ./

RUN npm install --prod

CMD node "$MAIN"
