#!groovy

node {

       load "$JENKINS_HOME/jobvars.env"

       dir('src/github.com/reportportal/service-analyzer') {

           stage('Checkout'){
                checkout scm
                sh 'git checkout master'
                sh 'git pull'
            }

            stage('Build') {
                 // Export environment variables pointing to the directory where Go was installed
                 docker.image('golang:1.8.3').inside("-u root -e GOPATH=${env.WORKSPACE}")  {
                        sh 'PATH=$PATH:$GOPATH/bin && make build v=`cat VERSION`-$BUILD_NUMBER'
                 }
                 archiveArtifacts artifacts: 'bin/*'
            }

           withEnv(["IMAGE_POSTFIX=-dev"]) {
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