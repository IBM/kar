ARG JS_RUNTIME=us.icr.io/research/kar-dev/sdk-nodejs-v12

FROM $JS_RUNTIME

WORKDIR /kar/app

# Default is to run philosophers.js
ENV MAIN=philosophers.js

# Copy dependencies first; they change rarely
COPY package.json package-lock.json yalc.lock ./
COPY .yalc .yalc
RUN npm install --prod

# Copy in application source
COPY *.js ./
