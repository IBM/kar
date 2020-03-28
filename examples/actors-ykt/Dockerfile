FROM node:12-alpine

ENV MAIN ykt.js

WORKDIR /kar/sdk/js

COPY sdk/js/*.js sdk/js/*.json ./

WORKDIR /kar/examples/actors-ykt

COPY examples/actors-ykt/*.js examples/actors-ykt/*.json ./

RUN npm install --prod

CMD node "$MAIN"
