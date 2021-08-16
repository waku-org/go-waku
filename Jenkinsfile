pipeline {
  agent {
    label 'linux'
  }

  options {
    timestamps()
    buildDiscarder(logRotator(
      numToKeepStr: '10',
      daysToKeepStr: '30',
    ))
  }

  environment {
    GOPATH = "${env.HOME}/go"
    PATH   = "${env.PATH}:${env.GOPATH}/bin"
  }

  stages {
    stage('Deps') {
      steps { sh 'make deps' }
    }

    stage('Lint') {
      steps { sh 'make lint' }
    }

    stage('Test') {
      steps { sh 'make test' }
    }
  }
  post {
    always { cleanWs() }
  }
}
