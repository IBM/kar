#!/bin/bash

kar -v debug -recv 30666 -app ykt -service client curl -X POST  http://localhost:30666/kar/actor-call/Floor/2/count
