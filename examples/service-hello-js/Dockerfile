ARG JS_RUNTIME=us.icr.io/research/kar-dev/sdk-nodejs-v12

FROM $JS_RUNTIME

WORKDIR /kar/app

# Default is to run server.js
ENV MAIN=server.js

# Copy dependencies first; they change rarely
COPY package.json package-lock.json ./
RUN npm install --prod

# Copy in application source
COPY *.js ./
