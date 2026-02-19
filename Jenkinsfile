// =============================================================================
// Civic Connect – Jenkins Pipeline (alternative to GitHub Actions)
// =============================================================================
// Requires: Docker, kubectl plugins on Jenkins agent
// =============================================================================

pipeline {
    agent any

    environment {
        REGISTRY     = 'ghcr.io'
        IMAGE_PREFIX = 'civic-connect'
        GIT_COMMIT   = sh(script: 'git rev-parse --short HEAD', returnStdout: true).trim()
    }

    stages {
        stage('Checkout') {
            steps {
                checkout scm
            }
        }

        stage('Test') {
            parallel {
                stage('Admin Service') {
                    steps {
                        dir('admin-service') {
                            sh 'go mod tidy && go vet ./...'
                        }
                    }
                }
                stage('Content Service') {
                    steps {
                        dir('content-service') {
                            sh 'npm ci'
                        }
                    }
                }
                stage('Complaint Service') {
                    steps {
                        dir('complaint-service') {
                            sh 'go mod tidy && go vet ./...'
                        }
                    }
                }
                stage('AI Worker') {
                    steps {
                        dir('ai-worker') {
                            sh 'pip install -r requirements.txt && python -m py_compile main.py'
                        }
                    }
                }
                stage('Chatbot Service') {
                    steps {
                        dir('chatbot-service') {
                            sh 'pip install -r requirements.txt && python -m py_compile main.py'
                        }
                    }
                }
            }
        }

        stage('Build Images') {
            steps {
                script {
                    def services = ['admin-service', 'content-service', 'complaint-service', 'ai-worker', 'chatbot-service', 'admin-panel']
                    services.each { svc ->
                        sh "docker build -t ${REGISTRY}/${IMAGE_PREFIX}/${svc}:${GIT_COMMIT} -f ${svc}/Dockerfile ."
                        sh "docker tag ${REGISTRY}/${IMAGE_PREFIX}/${svc}:${GIT_COMMIT} ${REGISTRY}/${IMAGE_PREFIX}/${svc}:latest"
                    }
                }
            }
        }

        stage('Push Images') {
            when { branch 'main' }
            steps {
                withCredentials([string(credentialsId: 'ghcr-token', variable: 'GHCR_TOKEN')]) {
                    sh "echo ${GHCR_TOKEN} | docker login ${REGISTRY} -u jenkins --password-stdin"
                }
                script {
                    def services = ['admin-service', 'content-service', 'complaint-service', 'ai-worker', 'chatbot-service', 'admin-panel']
                    services.each { svc ->
                        sh "docker push ${REGISTRY}/${IMAGE_PREFIX}/${svc}:${GIT_COMMIT}"
                        sh "docker push ${REGISTRY}/${IMAGE_PREFIX}/${svc}:latest"
                    }
                }
            }
        }

        stage('Deploy to K8s') {
            when { branch 'main' }
            steps {
                withKubeConfig([credentialsId: 'kubeconfig']) {
                    sh '''
                        kubectl apply -f infrastructure/k8s/namespace.yaml
                        kubectl apply -f infrastructure/k8s/secrets.yaml
                        kubectl apply -f infrastructure/k8s/postgres.yaml
                        kubectl apply -f infrastructure/k8s/rabbitmq.yaml
                        kubectl apply -f infrastructure/k8s/minio.yaml
                        kubectl apply -f infrastructure/k8s/redis.yaml
                        kubectl apply -f infrastructure/k8s/services.yaml
                        kubectl apply -f infrastructure/k8s/ingress.yaml
                    '''
                }
            }
        }
    }

    post {
        success { echo '✅ Pipeline completed successfully' }
        failure { echo '❌ Pipeline failed' }
    }
}
