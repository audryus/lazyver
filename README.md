# lazyver
Version generator for lazy people.

It'll create a new file in the path
if no Path is used, then the current path is consider.

```yaml
major: 1
minor: 2
patch: 3
last: 2024-10-11T09:19:47.01-03:00
kind: lazy
```

# Usage

Can't mix the two kinds. If needed delete the genertate file (.lazyver.yaml).

## Install
> go install github.com/audryus/lazyver

## Lazy
> lazyver lazy --path D:\projs\proj_with_git\ -o

## Semver
> lazyver semver --path D:\projs\proj_with_git\ -o

## Params
> --path 
>> Path with .git in it

> -o 
>> output the version
