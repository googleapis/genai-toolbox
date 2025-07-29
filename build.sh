#!/bin/bash

# Build and push script for genai-toolbox Docker image

set -e

# Configuration
AWS_REGION="us-east-2"
AWS_ACCOUNT_ID="381491897823"
ECR_REPOSITORY="anetac/dev/genai-toolbox"
IMAGE_TAG="latest"
FULL_IMAGE_NAME="${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com/${ECR_REPOSITORY}:${IMAGE_TAG}"

echo "🚀 Building genai-toolbox Docker image..."

# Build the Docker image
docker build -t ${ECR_REPOSITORY}:${IMAGE_TAG} .

echo "✅ Docker image built successfully"

# Tag the image for ECR
docker tag ${ECR_REPOSITORY}:${IMAGE_TAG} ${FULL_IMAGE_NAME}

echo "🏷️  Image tagged for ECR: ${FULL_IMAGE_NAME}"

# Login to ECR
echo "🔐 Logging into ECR..."
aws ecr get-login-password --region ${AWS_REGION} | docker login --username AWS --password-stdin ${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com

# Push the image to ECR
echo "📤 Pushing image to ECR..."
docker push ${FULL_IMAGE_NAME}

echo "✅ Image pushed successfully to ECR!"
echo "🎯 Image URI: ${FULL_IMAGE_NAME}"
echo ""
echo "📋 To deploy to Kubernetes, run:"
echo "   kubectl apply -f ../deploy-genai-toolbox.yaml" 
