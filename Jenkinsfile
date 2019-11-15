rrpBuildGoCode {
    projectKey = 'loss-prevention-service'
    dockerBuildOptions = ['--squash', '--build-arg GIT_COMMIT=$GIT_COMMIT']
    testStepsInParallel = false
    buildImage = 'amr-registry.caas.intel.com/rrp/ci-go-build-image:1.12.0-alpine'
    dockerImageName = "rsp/${projectKey}"
    ecrRegistry = "280211473891.dkr.ecr.us-west-2.amazonaws.com"
    protexProjectName = 'bb-loss-prevention-service'
    skipTests = true

    customBuildScript = 'make clean build force-test'

    infra = [
        stackName: 'RSP-Codepipeline-loss-prevention-service'
    ]

    notify = [
        slack: [ success: '#ima-build-success', failure: '#ima-build-failed' ]
    ]
}
