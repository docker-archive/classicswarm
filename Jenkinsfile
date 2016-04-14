// Only run on Linux atm
wrappedNode(label: 'docker') {
  deleteDir()
  stage "checkout"
  checkout scm

  documentationChecker("docs")
}

def testTask(imageName, imageTag) {
  return {
    wrappedNode(label: "linux && aufs") {
      deleteDir()
      checkout scm
      withChownWorkspace {
        withEnv(["DOCKER_IMAGE=${imageName}", "DOCKER_VERSION=${imageTag}"]) {
          sh """
          export STORAGE_DRIVER=\$( docker info | awk -F ': ' '\$1 == "Storage Driver" { print \$2; exit }' )
          docker pull dockerswarm/swarm-test-env:latest
          docker run --rm \\
          -i \\
          --privileged \\
          -e DOCKER_IMAGE \\
          -e DOCKER_VERSION \\
          -e STORAGE_DRIVER \\
          -v "\$(pwd):/go/src/github.com/docker/swarm" \\
          dockerswarm/swarm-test-env:latest ./test_runner.sh
          """
        }
      }
    }
  }
}

stage "test"
parallel([
  failFast: false,
  "1.9.1": this.testTask("dockerswarm/dind", "1.9.1"),
  "1.10.3": this.testTask("dockerswarm/dind", "1.10.3"),
  "1.11.2": this.testTask("dockerswarm/dind", "1.11.2"),
  "1.12.1": this.testTask("dockerswarm/dind", "1.12.1"),
  "master": this.testTask("dockerswarm/dind-master", "latest"),
])
