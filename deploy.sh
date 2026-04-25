#!/bin/bash

# GCP Free Tier Deployment Script
# This script deploys the auth-service to GCP Cloud Run completely free

set -e

echo "🚀 Auth Service - GCP Free Tier Deployment"
echo "=========================================="

# Check prerequisites
if ! command -v gcloud &> /dev/null; then
    echo "❌ gcloud CLI not installed. Install from: https://cloud.google.com/sdk/docs/install"
    exit 1
fi

# Step 1: Get or Create GCP Project
echo ""
echo "📦 Step 1: Setting up GCP Project..."
PROJECT_NAME="auth-service-demo"

# Check if project already exists
EXISTING_PROJECT=$(gcloud projects list --format='value(projectId)' --filter="name:$PROJECT_NAME" | head -1)
if [ -n "$EXISTING_PROJECT" ]; then
    PROJECT_ID=$EXISTING_PROJECT
    echo "✅ Using existing project: $PROJECT_ID"
else
    PROJECT_ID="${PROJECT_NAME}-$(date +%s | tail -c 6)"
    gcloud projects create $PROJECT_ID --name="$PROJECT_NAME"
    echo "✅ Project created: $PROJECT_ID"
fi

gcloud config set project $PROJECT_ID

# Step 1b: Link Billing Account (if not already linked)
echo ""
echo "💳 Step 1b: Ensuring billing is enabled..."
BILLING_STATUS=$(gcloud billing projects describe $PROJECT_ID --format='value(billingEnabled)' 2>/dev/null || echo "false")
if [ "$BILLING_STATUS" != "True" ]; then
    BILLING_ACCOUNT=$(gcloud billing accounts list --format='value(name)' --filter='open=true' | head -1)
    if [ -z "$BILLING_ACCOUNT" ]; then
        echo "❌ No active billing account found. Please enable billing at https://console.cloud.google.com/billing"
        exit 1
    fi
    gcloud billing projects link $PROJECT_ID --billing-account=$BILLING_ACCOUNT || true
    echo "✅ Billing account linked"
else
    echo "✅ Billing already enabled"
fi

# Step 2: Enable APIs
echo ""
echo "🔌 Step 2: Enabling required APIs..."
gcloud services enable \
  run.googleapis.com \
  cloudbuild.googleapis.com \
  artifactregistry.googleapis.com \
  secretmanager.googleapis.com \
  iam.googleapis.com \
  cloudresourcemanager.googleapis.com
echo "✅ APIs enabled"

# Step 3: Create Artifact Registry
echo ""
echo "📦 Step 3: Creating Artifact Registry..."
gcloud artifacts repositories create auth-service \
  --repository-format=docker \
  --location=us-central1 \
  --description="Auth service container images"
echo "✅ Artifact Registry created"

# Step 4: Setup Service Accounts
echo ""
echo "👤 Step 4: Setting up Service Accounts..."
PROJECT_NUMBER=$(gcloud projects describe $PROJECT_ID --format='value(projectNumber)')

# Grant Cloud Build permissions
gcloud projects add-iam-policy-binding $PROJECT_ID \
  --member="serviceAccount:$PROJECT_NUMBER@cloudbuild.gserviceaccount.com" \
  --role="roles/run.admin" \
  --quiet

gcloud projects add-iam-policy-binding $PROJECT_ID \
  --member="serviceAccount:$PROJECT_NUMBER@cloudbuild.gserviceaccount.com" \
  --role="roles/iam.serviceAccountUser" \
  --quiet

gcloud projects add-iam-policy-binding $PROJECT_ID \
  --member="serviceAccount:$PROJECT_NUMBER@cloudbuild.gserviceaccount.com" \
  --role="roles/artifactregistry.writer" \
  --quiet

# Create runtime service account
gcloud iam service-accounts create auth-service-runtime \
  --display-name="Auth Service Runtime" \
  --quiet

gcloud projects add-iam-policy-binding $PROJECT_ID \
  --member="serviceAccount:auth-service-runtime@$PROJECT_ID.iam.gserviceaccount.com" \
  --role="roles/secretmanager.secretAccessor" \
  --quiet

echo "✅ Service accounts configured"

# Step 5: Create Secrets
echo ""
echo "🔐 Step 5: Creating secrets in Secret Manager..."
echo "⚠️  You need to provide these values:"
echo ""

read -p "Enter PostgreSQL connection string (from Neon): " DB_URL
read -p "Enter Redis URL (from Redis Cloud): " REDIS_URL
read -p "Enter Google OAuth Client ID: " GOOGLE_CLIENT_ID
read -p "Enter Google OAuth Client Secret: " GOOGLE_CLIENT_SECRET

# Generate JWT secrets
JWT_ACCESS_SECRET=$(openssl rand -base64 32)
JWT_REFRESH_SECRET=$(openssl rand -base64 32)

# Create secrets
echo -n "$DB_URL" | gcloud secrets create db-url --data-file=- --quiet
echo -n "$REDIS_URL" | gcloud secrets create redis-url --data-file=- --quiet
echo -n "$JWT_ACCESS_SECRET" | gcloud secrets create jwt-access-secret --data-file=- --quiet
echo -n "$JWT_REFRESH_SECRET" | gcloud secrets create jwt-refresh-secret --data-file=- --quiet
echo -n "$GOOGLE_CLIENT_ID" | gcloud secrets create google-client-id --data-file=- --quiet
echo -n "$GOOGLE_CLIENT_SECRET" | gcloud secrets create google-client-secret --data-file=- --quiet

# Grant Cloud Build access to secrets
for secret in db-url redis-url jwt-access-secret jwt-refresh-secret google-client-id google-client-secret; do
  gcloud secrets add-iam-policy-binding $secret \
    --member="serviceAccount:$PROJECT_NUMBER@cloudbuild.gserviceaccount.com" \
    --role="roles/secretmanager.secretAccessor" \
    --quiet
done

echo "✅ Secrets created and configured"

# Step 6: Deploy with Cloud Build
echo ""
echo "🚀 Step 6: Deploying to Cloud Run..."
echo "This may take 2-3 minutes..."

gcloud builds submit --config=.gcp/cloudbuild.yaml

# Step 7: Get Cloud Run URL
echo ""
echo "✅ Deployment complete!"
echo ""
SERVICE_URL=$(gcloud run services describe auth-service \
  --region=us-central1 \
  --format='value(status.url)')

echo "📍 Your service is live at: $SERVICE_URL"
echo ""
echo "💡 Next steps:"
echo "1. Test your API:"
echo "   curl -X POST $SERVICE_URL/auth/register \\"
echo "     -H 'Content-Type: application/json' \\"
echo "     -d '{\"email\":\"test@example.com\",\"password\":\"Test123!\",\"password_confirm\":\"Test123!\"}'"
echo ""
echo "2. Make it public (optional):"
echo "   gcloud run services add-iam-policy-binding auth-service \\"
echo "     --region=us-central1 --member='allUsers' --role='roles/run.invoker'"
echo ""
echo "3. View logs:"
echo "   gcloud logging read \"resource.service_name=auth-service\" --limit 50 --format=json"
echo ""
echo "🎉 You're running on GCP completely FREE!"
