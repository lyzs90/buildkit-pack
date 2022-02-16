# buildkit-pack

Generic BuildKit frontend for building [Buildpacks](https://buildpacks.io/) directly. Adapted from [tonistiigi/buildkit-pack](https://github.com/tonistiigi/buildkit-pack).

## Improvements
- Uses the Cloud Native Buildpacks [pack](https://github.com/buildpacks/pack) CLI under the hood.
- Allows you to specify a builder and use a project descriptor to pass in build env vars.

## Usage

### With `buildctl`:
```sh
buildctl build --frontend=gateway.v0 --opt source=lyzs90/pack --local context=. --opt build-arg:builder=cnbs/sample-builder:bionic
```

## Options

`--opt build-arg:builder=<builder-img>`

## Examples