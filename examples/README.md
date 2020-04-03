### Example Structure

Each child directory contains an example KAR application.

Each example contains a `deploy` directory that contains
artifacts for deploying the application. Multiple deployment modes may
be supported via a combination of scripts, yaml files, and Helm
charts.  See the README.md in each directory for instructions.

### Examples in a Nutshell

+ [Hello World](./helloWorld) extends a simple
  [NodeJS Express server](helloWorld/server.js) to work with KAR.

+ [Yorktown Simulation](./actors-ykt) demonstrates using KAR's agent
  model for virtual stateful services to implement a scalable simulation.

+ (Unit Tests](./unit-tests) contains unit tests and scripts to
  exeucte them.
