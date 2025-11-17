# Deployment & Infrastructure

## Hosting Platform
- **Platform**: Heroku (https://www.heroku.com)
  - Platform-as-a-Service (PaaS) for easy deployment and management
  - Automatic SSL certificates
  - Built-in CI/CD with Git integration
  - Easy scaling (horizontal and vertical)
  - Add-ons ecosystem for databases, monitoring, etc.

## Heroku Configuration

### Application Setup
- **Buildpack**: Heroku Go buildpack (heroku/go)
  - Automatically detects Go applications via `go.mod`
  - Compiles binary during deployment
  - Runs compiled binary on dyno startup
- **Procfile**: Required file defining how to run the application
  ```
  web: bin/kendalls-nails-api
  ```
- **Port Binding**: Application must bind to `$PORT` environment variable (provided by Heroku)
- **Dyno Type**:
  - **Development/Staging**: Hobby dyno ($7/month)
  - **Production**: Standard or Performance dyno (based on traffic)

### Database Add-on
- **Add-on**: Heroku Postgres (https://www.heroku.com/postgres)
  - Managed PostgreSQL database
  - Automatic backups
  - High availability options
  - Connection pooling built-in
- **Plan Tiers**:
  - **Development**: Mini ($5/month, 10K rows limit) - for testing only
  - **Staging**: Basic ($9/month, 10M rows limit)
  - **Production**: Standard-0 or higher ($50+/month, unlimited rows)
- **Connection String**: Provided via `DATABASE_URL` environment variable
  - Format: `postgres://user:password@host:port/database`
  - GORM connects using this URL directly
- **Connection Pooling**: Configure max connections based on plan limits
  - Hobby/Basic plans: Max 20 connections
  - Standard plans: Max 120 connections
  - Adjust GORM pool settings accordingly (see Database section)

### Environment Variables
All configuration stored in Heroku config vars (environment variables):

**Required Variables:**
- `PORT` - Provided by Heroku automatically
- `DATABASE_URL` - Provided by Heroku Postgres add-on
- `AUTH0_DOMAIN` - Auth0 tenant domain (e.g., `yourapp.us.auth0.com`)
- `AUTH0_AUDIENCE` - Auth0 API identifier
- `AUTH0_CLIENT_ID` - Auth0 application client ID
- `AUTH0_CLIENT_SECRET` - Auth0 application client secret (for Management API)
- `AWS_ACCESS_KEY_ID` - AWS credentials for S3
- `AWS_SECRET_ACCESS_KEY` - AWS secret key for S3
- `AWS_REGION` - AWS region (e.g., `us-east-1`)
- `S3_BUCKET_NAME` - S3 bucket name (e.g., `kendalls-nails-prod`)
- `CORS_ALLOWED_ORIGINS` - Comma-separated list of allowed frontend URLs
- `ENVIRONMENT` - Current environment (`development`, `staging`, `production`)

**Optional Variables:**
- `LOG_LEVEL` - Logging level (`debug`, `info`, `warn`, `error`)
- `RATE_LIMIT_ENABLED` - Enable/disable rate limiting (`true`/`false`)
- `MAX_UPLOAD_SIZE_MB` - Max file upload size in MB (default: 10)

**Setting Config Vars:**
```bash
heroku config:set AUTH0_DOMAIN=yourapp.us.auth0.com
heroku config:set AWS_ACCESS_KEY_ID=your_access_key
```

### Deployment Process

**Initial Setup:**
1. Create Heroku application: `heroku create kendalls-nails-api`
2. Add Postgres add-on: `heroku addons:create heroku-postgresql:mini`
3. Set all environment variables using `heroku config:set`
4. Configure buildpack: `heroku buildpacks:set heroku/go`

**Deployment Flow:**
1. Commit code to Git repository
2. Push to Heroku remote: `git push heroku main`
3. Heroku automatically:
   - Detects Go application
   - Runs `go build` to compile binary
   - Places binary in `bin/` directory
   - Starts web dyno using Procfile command
4. Database migrations run automatically on startup (via GORM AutoMigrate in development)
5. Application becomes available at `https://kendalls-nails-api.herokuapp.com`

**Multiple Environments:**
- **Development**: Local machine or Heroku review apps
- **Staging**: Separate Heroku app (`kendalls-nails-staging`)
- **Production**: Main Heroku app (`kendalls-nails-api`)

Each environment has its own:
- Heroku app instance
- Postgres database
- S3 bucket
- Auth0 tenant (or separate applications within same tenant)
- Environment variables

### Scaling
**Horizontal Scaling (more dynos):**
```bash
heroku ps:scale web=2  # Run 2 web dynos
```

**Vertical Scaling (bigger dynos):**
```bash
heroku ps:resize web=standard-1x  # Upgrade to standard dyno
```

**Database Scaling:**
- Upgrade plan via Heroku dashboard or CLI
- Connection pooling handles increased load
- Add follower databases for read replicas (advanced)

### Health Checks
- Implement `/health` or `/api/v1/health` endpoint
- Returns 200 OK if application is healthy
- Heroku monitors this endpoint to detect crashes
- Include checks for:
  - Database connectivity
  - S3 connectivity
  - Auth0 connectivity

### SSL/TLS
- Automatic SSL certificates provided by Heroku
- All traffic encrypted via HTTPS
- Custom domain support (e.g., `api.kendallsnails.com`)
- Configure custom domain:
  ```bash
  heroku domains:add api.kendallsnails.com
  ```

### Logging
- Application logs via `stdout`/`stderr` (use Go's `log` package or structured logging)
- View logs: `heroku logs --tail`
- Log retention: 1,500 lines (Heroku default)
- For longer retention, add logging add-on:
  - Papertrail (free tier: 50MB/month search)
  - Logentries or other providers

### Backups
- **Database Backups**:
  - Automatic daily backups (included with Postgres Standard plans)
  - Manual backups: `heroku pg:backups:capture`
  - Restore backup: `heroku pg:backups:restore`
- **S3 Backups**:
  - S3 versioning can be enabled for image backups
  - Cross-region replication for disaster recovery (optional)

### CI/CD Integration
**Option 1: Heroku Pipelines**
- Create pipeline with dev → staging → production
- Enable review apps for pull requests
- Promote releases from staging to production

**Option 2: GitHub Actions + Heroku**
- GitHub Actions workflow builds and tests code
- On merge to main, automatically deploy to Heroku
- Example workflow file: `.github/workflows/deploy.yml`

### Cost Estimate (Monthly)
**Staging Environment:**
- Hobby dyno: $7
- Postgres Basic: $9
- Total: ~$16/month

**Production Environment:**
- Standard-1X dyno: $25
- Postgres Standard-0: $50
- S3 storage: ~$5 (depends on usage)
- Auth0 free tier: $0 (up to 7,000 active users)
- Total: ~$80/month (scales with usage)
