working_branch = 'master'


pipeline {
    agent {
        label 'DOCKER_V18'
    }
    triggers {
        pollSCM 'H/10 * * * *'
    }
    stages {
        stage('Checkout') {
            steps {
                checkout([$class: 'GitSCM', 
                branches: [[name: 'master']], 
                doGenerateSubmoduleConfigurations: false, 
                extensions: [[$class: 'SubmoduleOption',
                                    disableSubmodules: false,
                                    parentCredentials: true,
                                    recursiveSubmodules: true,
                                    reference: '',
                                    trackingSubmodules: false]], 
                submoduleCfg: [], 
                userRemoteConfigs: 
                    [[credentialsId: 'd20e86b7-33c4-40b8-8416-43ac83dec40d', url:'git@10.42.4.1:cyberarmor/k8s-ca-dashboard-aggregator.git']]])

            }
        }
        stage('Build') {
            steps {
                sh '''
                        docker build -t dreg.eust0.cyberarmorsoft.com:443/k8s-ca-dashboard-aggregator:v$BUILD_NUMBER -t dreg.eust0.cyberarmorsoft.com:443/k8s-ca-dashboard-aggregator:latest .
                '''
            }
        }
        stage('Upload') {
            steps {
                sh '''
                        docker push dreg.eust0.cyberarmorsoft.com:443/k8s-ca-dashboard-aggregator:v$BUILD_NUMBER 
                        docker push dreg.eust0.cyberarmorsoft.com:443/k8s-ca-dashboard-aggregator:latest
                '''
            }
        }

        
   } // stages

    post {  
        failure {  
            mail bcc: '', body: "<b>Pipline failure</b><br>Project: ${env.JOB_NAME} <br>Build Number: ${env.BUILD_NUMBER} <br> build URL: ${env.BUILD_URL}", cc: '', charset: 'UTF-8', from: 'jenkins@cyberarmor.io', mimeType: 'text/html', replyTo: 'no-reply@cyberarmor.io', subject: "${env.JOB_NAME} failed!", to: "development@cyberarmor.io";  
        }  
        fixed {  
            mail bcc: '', body: "<b>Pipline state resumed</b><br>Project: ${env.JOB_NAME} <br>Build Number: ${env.BUILD_NUMBER} <br> build URL: ${env.BUILD_URL}", cc: '', charset: 'UTF-8', from: 'jenkins@cyberarmor.io', mimeType: 'text/html', replyTo: 'no-reply@cyberarmor.io', subject: "${env.JOB_NAME} pipeline fixed", to: "development@cyberarmor.io";  
        }  
    }

}
