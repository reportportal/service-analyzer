#!groovy

node {

    load "$JENKINS_HOME/jobvars.env"

    dir('src/github.com/reportportal/service-analyzer') {

        stage('Checkout') {
            checkout scm
            sh 'git checkout master'
            sh 'git pull'
        }

        stage('Build') {
            withEnv(["IMAGE_POSTFIX=-dev", "VERSION=`cat VERSION`-$BUILD_NUMBER"]) {
                docker.withServer("$DOCKER_HOST") {
                    stage('Build Docker Image') {
                        sh 'make build-image'
                    }

                    stage('Deploy container') {
                        sh "docker-compose -p reportportal -f $COMPOSE_FILE up -d --force-recreate analyzer"
                    }
                }
            }
        }

    }
}