FROM node:12-alpine

ENV MAIN server.js

WORKDIR /kar-sample

ADD *.js ./
COPY package.json .
COPY package-lock.json .

RUN npm install

CMD node "$MAIN"
