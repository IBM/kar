# Making a KAR Release

## Release Contents

A release of the core KAR system generates a number of artifacts:

+ A source release (github release page)
+ Prebuilt `kar` cli (github release page)
+ Tagged container images (quay.io)
+ Java SDK: com.ibm.research.kar pom/jars (central repository)
+ JavaScript SDK: kar-sdk (npmjs)
+ Helm chart for deploying kar (gh-pages branch)
+ Swagger specification (gh-pages branch)

## Release Procedures

Locally create a branch "release-prep"

### Update CHANGELOG.md

1. Summarize non-trivial changes from `git log` into CHANGELOG.md
2. Commit update CHANGELOG.md to release-prep

### Prepare and publish SDKs

#### Java SDK

1. Bump version `mvn versions:set -DnewVersion=x.y.z`
2. Publish to The Central Repository
    + `mvn clean deploy -P release`
    + `mvn nexus-staging:close -DstagingRepositoryId=comibmresearchkar-NNNN`
    + `mvn nexus-staging:release -DstagingRepositoryId=comibmresearchkar-NNNN`
3. Commit version bump to release-prep

#### JavaScript SDK

1. update `version` in package.json and package-lock.json
2. Commit version bump to release-prep
3. Publish to npmjs
   + `npm login`
   + `npm publish --dry-run`
   + Verify that what will be being published looks right:
```
npm notice 📦  kar-sdk@0.6.0
npm notice === Tarball Contents ===
npm notice 11.4kB LICENSE.txt
npm notice 685B   README.md
npm notice 14.3kB index.d.ts
npm notice 17.2kB index.js
npm notice 728B   package.json
npm notice === Tarball Details ===
npm notice name:          kar-sdk
npm notice version:       0.6.0
npm notice filename:      kar-sdk-0.6.0.tgz
npm notice package size:  11.1 kB
npm notice unpacked size: 44.3 kB
npm notice shasum:        2aa258e5df3b80b847d23fe88e62d2c42e7b5284
npm notice integrity:     sha512-Bjj8oYJ1Ghn3o[...]bQXNEgOyrsCzg==
npm notice total files:   5
```
   + `npm publish --public`

#### Python SDK

1. Update `version` in `setup.py` file.
2. Commit version bump to release-prep
3. Prepare environment for package publishing (first time step only):
```
python3 -m pip install --upgrade build
python3 -m pip install --upgrade twine
```
4. Build the PyPi package by running the following command inside the `sdk-python` subdirectory of KAR. The command will create a `dist` subdirectory with the package files.
```
python3 -m build
```
5. Publish the contents of the dist subdirectory to PyPi:
```
python3 -m twine upload --repository pypi dist/*
```

### Publish Helm Chart

1. Update version numbers in:
    + scripts/helm/kar/Chart.yaml
    + scripts/helm/kar/values.yaml
2. Commit change to release-prep
3. `helm package scripts/helm/kar`
4. switch to gh-pages branch
5. copy in kar-x.y.z.tgz
5. `helm repo index charts --url https://ibm.github.io/kar/charts`
6. Commit updated index.yaml and new kar-x.y.z.tgz to gh-pages
7. switch back to release-prep branch

### Update examples to use new SDKs

1. examples/*java*
   + pom.xml: update version.kar-java-sdk to x.y.z
   + `mvn versions:set -DnewVersion=x.y.z`

2. examples/*js* benchmark/*
   + update package.json and package-lock.json

3. Commit changes to release-prep.

### PR & Merge the release-prep branch

1. PR the release-prep branch. Travis should pass. Merge PR.

### Tag repository

1. `git tag -s vx.y.z`
   `git tag -s core/vx.y.z`
2. `git push --tags upstream`
3. Tags starting with `v` trigger build processes that:
    * push tagged images to quay.io (via travis-ci)
    * make a github release with source and cli tarballs (via github actions)
   Monitor to make sure all of these builds are successful.
4. Update the git release with the CHANGELOG

### Cleanup

1. Change Java-SDK pom to x.y.(z+1)-SNAPSHOT
`mvn versions:set -DnewVersion=x.y.(z+1)-SNAPSHOT`

2. PR version changes to main.

