# Build Docker image
docker build -t own-your-music .

# Login to ECR
aws ecr get-login-password --region us-east-2 | docker login --username AWS --password-stdin 181179258922.dkr.ecr.us-east-2.amazonaws.com

# Tag and push to ECR
docker tag own-your-music:latest 181179258922.dkr.ecr.us-east-2.amazonaws.com/own-your-music:latest
docker push 181179258922.dkr.ecr.us-east-2.amazonaws.com/own-your-music:latest

# Update Lambda function with new image
aws lambda update-function-code --function-name $LAMBDA_FUNCTION_NAME --image-uri 181179258922.dkr.ecr.us-east-2.amazonaws.com/own-your-music:latest --region us-east-2