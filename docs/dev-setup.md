# Notes for KAR Developers

This file collects various hints and tips that are useful to
people who are developing the KAR runtime system, but are not
relevant to using KAR to build applications.

### Local Development - JavaScript SDK - Yalc

We use [yalc](https://www.npmjs.com/package/yalc) to keep the example packages
and the JavaScript SDK package in sync. When making and testing local changes to
the JavaScript SDK these changes need to be propagated to the examples projects
using `yalc`. First install `yalc`:
```shell
$ npm i -g yalc
```
Then configure `yalc` for `KAR`:
```shell
./scripts/setup-yalc.sh
```
Finally, whenever a change is made to the JavaScript SDK run:
```shell
cd sdk-js
yalc push
```

### Local Development - Running test cases

The scripts in the `travis` directory are a good way
execute test cases during development.
