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
                        sh 'make build-image-dev v=`cat VERSION`-$BUILD_NUMBER'
                    }

                    stage('Deploy container') {
                        sh "docker-compose -p reportportal5 -f $COMPOSE_FILE_RP_5 up -d --force-recreate analyzer"
                    }
                }
            }
        }

    }
}