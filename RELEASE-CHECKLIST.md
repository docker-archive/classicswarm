# Release Checklist

### 1. Update version and CHANGELOG.md on docker/swarm

```
git checkout -b bump-<version>
edit version/version.go
edit CHANGELOG.md
git add .
git commit -s -m "Bump version to <version>"
git push $GITHUBUSER bump-<version>
```

Open PR on docker/swarm. Merge the PR before moving to next step.

### 2. Rebase release branch on top of master branch and tag

```
git checkout release
git rebase master
git push origin
git tag <tag>
git push origin <tag>
```

### 3. Update library image

If you don't have `github.com:docker/swarm-library-image.git` repo, clone it.

```
git clone git@github.com:docker/swarm-library-image.git
```

Build and update Swarm image.

```
cd swarm-library-image
git pull
./update.sh <tag> (example: ./update.sh v0.2.0-rc2)
check build is successful (swarm binary should show in git diff)
git add .
git commit -s -m “<tag>"
git push origin
```

### 4. Update official image

If you don't have https://github.com/docker-library/official-images.git, fork it.

```
git clone https://github.com/docker-library/official-images.git
cd official-images
git remote add $GITHUBUSER git@github.com:$GITHUBUSER/official-images.git
```

Update Swarm image list. You may add a new RC or a stable image to the image list. `latest` is for
latest stable image, normally an RC doesn't qualify.

```
git checkout master
git pull
git checkout -b update_swarm_<tag>
edit library/swarm
git add library/swarm
git commit -s -m "update swarm <tag>"
git push $GITHUBUSER update_swarm_<tag>
```

Open PR on docker-library/official-images.

### 5. Create release on github

Go to https://github.com/docker/swarm/releases/new use <tag> and changelog
