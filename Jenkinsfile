#!groovy

node {

    load "$JENKINS_HOME/jobvars.env"

    dir('src/github.com/reportportal/service-analyzer') {

        stage('Checkout') {
            checkout scm
        }

        stage('Build') {
            withEnv(["IMAGE_POSTFIX=-dev", "BUILD_NUMBER=${env.BUILD_NUMBER}"]) {
                docker.withServer("$DOCKER_HOST") {
                    stage('Build Docker Image') {
                        sh """
                            MAJOR_VER=\$(cat VERSION)
                            BUILD_VER="\${MAJOR_VER}-${env.BUILD_NUMBER}"
                            make get-build-deps build build-image-dev v=\$BUILD_VER
                        """
                    }

                    stage('Deploy container') {
                        sh "docker-compose -p reportportal5 -f $COMPOSE_FILE_RP_5 up -d --force-recreate analyzer"
                    }
                }
            }
        }

    }
}