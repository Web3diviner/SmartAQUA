# 🚀 SmartAQUA & AquaDoc AI — Production Deployment Guide

This guide details how to deploy **SmartAQUA & AquaDoc AI** to production in under **5 minutes** using **Render / Railway** (Backend) and **Vercel / Netlify** (Farmer App & Admin Portal).

---

## 🏗️ Architecture in Production

```
┌─────────────────────────────────────────────────────────────┐
│                 Vercel / Netlify (Edge CDN)                 │
│  ├─ Farmer Web App:  https://app.smartaqua.ai  (or .vercel) │
│  └─ Admin Portal:    https://admin.smartaqua.ai(or .vercel) │
└──────────────────────────────┬──────────────────────────────┘
                               │ HTTPS / JSON API
                               ▼
┌─────────────────────────────────────────────────────────────┐
│                 Render / Railway (PaaS Host)                │
│  └─ AquaDoc AI Backend: https://api.smartaqua.ai (onrender) │
│     ├─ Groq LPU Ultra-Fast LLM (Llama 3.3 / GPT-OSS 120B)   │
│     ├─ Whisper v3 Voice Note Transcription                  │
│     └─ 16 Verified West African Aquaculture Manuals (RAG)   │
└─────────────────────────────────────────────────────────────┘
```

---

## Part 1: Deploy Backend to Render (2 Minutes)

Render provides free or standard Docker hosting with automated HTTPS.

### Step 1: Connect GitHub Repo on Render
1. Go to **[dashboard.render.com](https://dashboard.render.com)** and sign in.
2. Click **"New +"** $\to$ **"Web Service"**.
3. Select your repository: **`Web3diviner/SmartAQUA`**.

### Step 2: Configure Web Service
- **Name:** `aquadoc-api` (or `smartaqua-api`)
- **Region:** Frankfurt (EU) or Oregon (US)
- **Language / Runtime:** `Docker`
- **Dockerfile Path:** `./aquadoc/Dockerfile`
- **Docker Context Path:** `./aquadoc`
- **Instance Type:** `Free` or `Starter ($7/mo)`

### Step 3: Add Environment Variables
Under the **Environment Variables** section, add:

| Key | Recommended Value | Description |
|---|---|---|
| `GROQ_API_KEY` | `gsk_...` *(Your Live Groq Key)* | Required for LPU LLM inference & Whisper transcription |
| `APP_ENV` | `development` *(or `production`)* | Set to `development` if using the built-in admin evaluation dashboard |
| `AQUADOC_DEV_TOKEN` | `aqua-prod-secret-2026` *(Pick a secret)* | Secure token for internal API access and admin traces |
| `CORS_ALLOW_ORIGINS` | `*` *(or your Vercel domains)* | Allows frontend web requests |
| `LLM_PROVIDER` | `groq` | Default high-speed provider |
| `LLM_MODEL` | `openai/gpt-oss-120b` *(or `meta-llama/llama-3.3-70b-versatile`)* | Primary reasoning model |
| `EMBEDDING_PROVIDER`| `hashing` | Zero-dependency high-speed vectorizer |

### Step 4: Click "Create Web Service"
Render will build the Docker container, index all 16 aquaculture manuals, and give you a live URL (e.g. `https://aquadoc-api.onrender.com`).

---

## Part 2: Deploy Farmer Web App to Vercel (1 Minute)

### Step 1: Import Project on Vercel
1. Go to **[vercel.com](https://vercel.com)** and click **"Add New..."** $\to$ **"Project"**.
2. Select repository: **`Web3diviner/SmartAQUA`**.

### Step 2: Configure Farmer App Settings
- **Project Name:** `smartaqua-web` (or `aquadoc-web`)
- **Framework Preset:** `Vite`
- **Root Directory:** Click **Edit** and choose `aquadoc-web`

### Step 3: Add Environment Variable
Under **Environment Variables**, add:
- `VITE_AQUADOC_BASE_URL` = `https://aquadoc-api.onrender.com` *(Your Render backend URL from Part 1)*

### Step 4: Click "Deploy"
Vercel will build the frontend and provide your live farmer URL (e.g., `https://smartaqua-web.vercel.app`).

---

## Part 3: Deploy Admin Operations Portal to Vercel (1 Minute)

### Step 1: Import Second Project on Vercel
1. On Vercel dashboard, click **"Add New..."** $\to$ **"Project"**.
2. Select the same repository: **`Web3diviner/SmartAQUA`**.

### Step 2: Configure Admin Settings
- **Project Name:** `smartaqua-admin`
- **Framework Preset:** `Vite`
- **Root Directory:** Click **Edit** and choose `aquadoc-admin`

### Step 3: Add Environment Variables
Under **Environment Variables**, add:
- `VITE_AQUADOC_BASE_URL` = `https://aquadoc-api.onrender.com`
- `VITE_AQUADOC_DEV_TOKEN` = `aqua-prod-secret-2026` *(Must match the `AQUADOC_DEV_TOKEN` on Render)*

### Step 4: Click "Deploy"
Your live enterprise admin portal is ready (e.g., `https://smartaqua-admin.vercel.app`).

---

## Part 4: Google Authentication Configuration (Production Setup)

To allow farmers to sign in with Google on your live production domain:
1. Go to **[Google Cloud Console — Credentials](https://console.cloud.google.com/apis/credentials)**.
2. Under **OAuth 2.0 Client IDs**, edit your Web client.
3. Under **Authorized JavaScript origins**, add:
   - `https://smartaqua-web.vercel.app`
   - *(Your custom domain if applicable, e.g., `https://app.smartaqua.ai`)*
4. Under **Authorized redirect URIs**, add:
   - `https://smartaqua-web.vercel.app`
   - `https://smartaqua-web.vercel.app/auth`
5. Click **Save**.

---

## Part 5: Production Verification Checklist

- [ ] **Backend Health Check:** Visit `https://your-api.onrender.com/internal/v1/health/live` (Should return `{"status": "ok"}`).
- [ ] **AI Veterinary Consultation:** Ask a question in the Farmer web chat. Verify sub-second streaming and verified citations.
- [ ] **Voice Note Transcription:** Record an audio question in the web app and verify Whisper transcription.
- [ ] **Consultant Booking:** Submit a test consultation booking. Verify it shows up in real time on `smartaqua-admin`.
- [ ] **Trace Inspector:** Verify live query latency and token metrics appear in the Admin Evaluation Hub.

---

*SmartAQUA Platform &copy; 2026. Empowering African Aquaculture.*
