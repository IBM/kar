#!/bin/sh

curl  -H "Content-Type: application/json" -X POST http://localhost:9090/Example/client/incrSync -d '10'
