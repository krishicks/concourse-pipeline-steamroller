# concourse-pipeline-steamroller

Convert this:

```yaml
jobs:
- name: my-job
  plan:
  - get: some-resource
  - task: some-task
    file: some-resource/tasks/some-task.yml
```

Into this:

```yaml
jobs:
- name: my-job
  plan:
  - get: some-resource
  - task: some-task
    config:
      platform: linux
      image_resource:
        type: docker-image
        source:
          repository: some-repository
      run:
        path: sh
        args:
        - -c
        - |
          #!/bin/bash

          echo 42
        dir: ""
      inputs:
      - name: some-resource
        path: ""
```

## Usage

```
go get github.com/krishicks/concourse-pipeline-steamroller/cmd/steamroll

cat > config.yml <<EOF
  resource_map:
    "some-resource": /path/to/resource/on/disk
EOF

steamroll -p some-pipeline.yml -c config.yml
```

See the example in `example/`. Run `scripts/create-example-config.sh` to make a
config.

## About

A common pattern when writing Concourse pipelines is to have tasks in the
pipeline refer to a task that comes from a resource (often, a git repository
somewhere). If you want to modify that task for some reason you can either
modify the task in the resource, commit and push the change, and then run your
pipeline again, or you can just inline the entire task directly into the
pipeline and just `fly set-pipeline`.

This does the inlining for you; it just needs you to have those tasks on disk
somewhere, and provide a config that maps resource names to absolute paths on
disk, such as:

```yaml
resource_map:
  "some-resource": /path/to/resource/on/disk
  "another-resource": /some/other/place/on/disk
```

It's pretty basic for now. It doesn't presently handle renamed resources, e.g.:

```yaml
- get: renamed-resource
  resource: actual-resource
- task: some-task
  file: renamed-resource/some-task.yml
```

If you use the above you'd need to use "renamed-resource" in your config to
tell it where "actual-resource" is because it just yanks the root
("renamed-resource") off of the `file:` directive to find the mapping in the
config.

It also doesn't remove the (now superfluous) `get:` step; but you'd probably
want to do that as it's no longer necessary and may lead to confusion. You
could use [`yaml-patch`](https://github.com/krishicks/yaml-patch) to do that
for you:

```yaml
- op: remove
  path: /jobs/get=renamed-resource
```
