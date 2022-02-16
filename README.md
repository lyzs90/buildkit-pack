# buildkit-pack

Generic BuildKit frontend for building [Buildpacks](https://buildpacks.io/) directly. Adapted from [tonistiigi/buildkit-pack](https://github.com/tonistiigi/buildkit-pack).

## Improvements
- Uses the Cloud Native Buildpacks [pack](https://github.com/buildpacks/pack) CLI under the hood.
- Allows you to specify a builder and use a project descriptor to pass in build env vars.


## Usage

### With `buildctl`:
```sh
buildctl build --frontend=gateway.v0 --opt source=lyzs90/pack --local context=. --builder=cnbs/sample-builder:bionic
```

### With Docker (v18.06+ with `DOCKER_BUILDKIT=1`):
Add `# syntax = lyzs90/pack` as the first line of the project descriptor file (eg. `project.toml`):
```sh
docker build -f project.toml . --builder=cnbs/sample-builder:bionic
```

## Options

## Examples