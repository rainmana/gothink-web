# GoThink MCP Server - Deployment Guide

This guide covers deploying the GoThink MCP server to various cloud platforms for remote access via HTTP/SSE transport.

## Quick Deploy

Click a button below to deploy to your preferred platform:

[![Deploy on Railway](https://railway.app/button.svg)](https://railway.app/template/new?template=https://github.com/rainmana/gothink-web)

[![Deploy to Render](https://render.com/images/deploy-to-render-button.svg)](https://render.com/deploy?repo=https://github.com/rainmana/gothink-web)

## Platform-Specific Instructions

### Railway

Railway offers the simplest deployment experience with automatic HTTPS.

**Step 1: Deploy**
1. Click the "Deploy on Railway" button above
2. Connect your GitHub account if prompted
3. Railway will automatically:
   - Build the Docker image
   - Deploy the service
   - Assign a public URL

**Step 2: Get Your URL**
1. Go to your Railway dashboard
2. Click on your gothink-mcp service
3. Go to "Settings" → "Domains"
4. Your service URL will be shown (e.g., `https://gothink-mcp-production.up.railway.app`)

**Step 3: Configure Environment Variables (Optional)**
1. Go to "Variables" tab
2. Add any custom environment variables from `.env.example`

**SSE Endpoint**: `https://your-app.up.railway.app/sse`

---

### Google Cloud Run

Google Cloud Run provides serverless container deployment with automatic scaling.

**Prerequisites**
- Google Cloud account
- `gcloud` CLI installed

**Step 1: Build and Push**
```bash
# Set your project ID
gcloud config set project YOUR_PROJECT_ID

# Build and push to Google Container Registry
gcloud builds submit --tag gcr.io/YOUR_PROJECT_ID/gothink-mcp

# Deploy to Cloud Run
gcloud run deploy gothink-mcp \
  --image gcr.io/YOUR_PROJECT_ID/gothink-mcp \
  --platform managed \
  --region us-central1 \
  --allow-unauthenticated \
  --port 8080
```

**Step 2: Get Your URL**
After deployment, Cloud Run will output your service URL.

**SSE Endpoint**: `https://gothink-mcp-xxxxx-uc.a.run.app/sse`

---

### Fly.io

Fly.io provides edge deployment with global distribution.

**Prerequisites**
- Fly.io account
- `flyctl` CLI installed

**Step 1: Login and Initialize**
```bash
# Login to Fly.io
flyctl auth login

# Initialize app (fly.toml already exists)
flyctl launch --no-deploy

# Update app name in fly.toml if needed
```

**Step 2: Deploy**
```bash
# Deploy the app
flyctl deploy

# Get your app URL
flyctl info
```

**Step 3: View Logs**
```bash
flyctl logs
```

**SSE Endpoint**: `https://gothink-mcp.fly.dev/sse`

---

### Render

Render provides simple container deployment with managed infrastructure.

**Step 1: Deploy**
1. Click the "Deploy to Render" button above
2. Connect your GitHub repository
3. Render will automatically detect the `render.yaml` Blueprint
4. Click "Apply" to create the service

**Step 2: Get Your URL**
1. Go to your Render dashboard
2. Click on your gothink-mcp service
3. Your service URL will be shown (e.g., `https://gothink-mcp.onrender.com`)

**Step 3: Configure Environment Variables (Optional)**
1. Go to "Environment" tab
2. Add any custom environment variables

**SSE Endpoint**: `https://gothink-mcp.onrender.com/sse`

---

## Environment Variables

Configure these environment variables on your platform:

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | HTTP server port (auto-set by most platforms) |
| `GOTHINK_HOST` | `0.0.0.0` | Host to bind to (use 0.0.0.0 for cloud) |
| `GOTHINK_LOG_LEVEL` | `info` | Logging level (debug, info, warn, error) |
| `GOTHINK_MAX_THOUGHTS_PER_SESSION` | `100` | Maximum thoughts per session |
| `GOTHINK_MENTAL_MODELS_PATH` | - | Custom mental models directory |

---

## Testing Your Deployment

### 1. Health Check

Test that your server is running:

```bash
curl https://your-deployment-url/health
```

Expected response:
```json
{
  "status": "healthy",
  "service": "gothink-mcp-server",
  "version": "1.0.0",
  "time": "2024-01-01T00:00:00Z"
}
```

### 2. Server Info

Check server information:

```bash
curl https://your-deployment-url/
```

### 3. SSE Connection

Test SSE endpoint (will keep connection open):

```bash
curl -N -H "Accept: text/event-stream" https://your-deployment-url/sse
```

---

## Connecting MCP Clients

### Claude Desktop

Add to your Claude Desktop configuration (`claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "gothink": {
      "transport": {
        "type": "sse",
        "url": "https://your-deployment-url/sse"
      }
    }
  }
}
```

### Other MCP Clients

Configure the SSE endpoint URL: `https://your-deployment-url/sse`

---

## Troubleshooting

### Server Won't Start

**Check Logs:**
- Railway: Dashboard → "Deployments" → Click deployment → "Logs"
- Cloud Run: `gcloud run logs read --service gothink-mcp --limit 50`
- Fly.io: `flyctl logs`
- Render: Dashboard → Service → "Logs"

**Common Issues:**
1. **Port binding**: Ensure `GOTHINK_HOST=0.0.0.0` is set
2. **Memory limits**: Increase if server is killed (OOMKilled)
3. **Build failures**: Check Docker build logs

### Health Check Failing

1. Verify port is correct (usually 8080 or set by `PORT` env var)
2. Check that `/health` endpoint is accessible
3. Ensure container is running: check platform dashboard

### SSE Connection Issues

1. **CORS errors**: The server includes CORS headers, but verify your client supports SSE
2. **Timeout**: Some platforms have connection timeouts - check platform docs
3. **SSL/TLS**: Ensure you're using `https://` URLs in production

### High Latency

1. **Choose region closer to you**: Deploy to a region geographically near your location
2. **Cold starts**: First request may be slow (especially on Google Cloud Run free tier)
3. **Check resource limits**: Increase CPU/memory allocation if needed

---

## Local Testing

Test the HTTP server locally before deploying:

```bash
# Build and run
make build-http
./gothink-http

# Or use Docker
docker build -t gothink-local .
docker run -p 8080:8080 -e PORT=8080 gothink-local

# Test endpoints
curl http://localhost:8080/health
curl -N -H "Accept: text/event-stream" http://localhost:8080/sse
```

---

## Cost Estimates

| Platform | Free Tier | Paid Tier |
|----------|-----------|-----------|
| **Railway** | $5 credit/month | ~$5-10/month for small app |
| **Google Cloud Run** | 2M requests/month free | Pay per request (~$0.40/M requests) |
| **Fly.io** | 3 shared-cpu VMs free | ~$3-5/month for dedicated |
| **Render** | Free tier available | $7/month starter plan |

> Most small deployments will fit within free tiers or cost less than $10/month.

---

## Security Considerations

1. **No built-in authentication**: The server is publicly accessible. For production:
   - Add API key authentication
   - Use platform authentication features
   - Restrict access via firewall rules

2. **Rate limiting**: Consider adding rate limiting for public deployments

3. **HTTPS only**: Always use HTTPS URLs in production (platforms provide this automatically)

---

## Monitoring

### Railway
- Dashboard shows CPU, memory, and request metrics
- Custom metrics via Railway observability

### Google Cloud Run
- Cloud Monitoring provides detailed metrics
- Set up alerts for errors and latency

### Fly.io
- Built-in metrics dashboard
- Grafana integration available

### Render
- Metrics tab shows resource usage
- Can connect external monitoring services

---

## Updating Your Deployment

### Railway / Render
Push to your GitHub repository - auto-deploys on push

### Google Cloud Run
```bash
gcloud builds submit --tag gcr.io/YOUR_PROJECT_ID/gothink-mcp
gcloud run deploy gothink-mcp --image gcr.io/YOUR_PROJECT_ID/gothink-mcp
```

### Fly.io
```bash
flyctl deploy
```

---

## Support

For issues or questions:
- GitHub Issues: https://github.com/rainmana/gothink-web/issues
- Documentation: https://github.com/rainmana/gothink-web
