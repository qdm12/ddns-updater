# Cloudflare DDNS-Updater Guide

This guide provides step-by-step instructions for setting up DDNS-Updater with Cloudflare. This solution will automatically update your Cloudflare DNS record when your public IP changes and provide a monitoring dashboard.

## Prerequisites

- Server with Docker installed
- Sudo/root access to the server
- Cloudflare account with API token
- Domain managed by Cloudflare (e.g., example.com)

## Step 1: Create a Cloudflare API Token

1. Log in to your Cloudflare dashboard at https://dash.cloudflare.com
2. Go to "My Profile" > "API Tokens" > "Create Token"
3. Select "Create Custom Token"
4. Name it "DDNS Updater"
5. Under "Permissions":
   - Zone - DNS - Edit
   - Zone - Zone - Read
6. Under "Zone Resources":
   - Include - Specific zone - your domain (e.g., example.com)
7. **Important**: Set "TTL" to "No expiration" (do not select a date range) or your token will expire and break your DDNS
8. Click "Continue to summary" then "Create Token"
9. Copy the generated token (you'll only see it once)

## Step 2: Prepare Your Docker Environment

```bash
# Create a directory for DDNS-Updater
mkdir -p ~/ddns-updater
cd ~/ddns-updater
```

This directory (`~/ddns-updater`) will be mounted into the container and will store your configuration file and update history. The important paths are:

- Host path: `~/ddns-updater/config.json` (expands to `/home/yourusername/ddns-updater/config.json`)
- Container path: `/updater/data/config.json`

## Step 3: Start the DDNS-Updater (Confirmed Working Method)

Based on extensive testing, this is the configuration that reliably works:

### Using a config.json file (CONFIRMED WORKING)

This method works with both the latest version and v2.5.0:

```bash
# Create directory for configuration
mkdir -p ~/ddns-updater

# Create the config.json file with your API token
cat > ~/ddns-updater/config.json << 'EOF'
{
  "settings": [
    {
      "provider": "cloudflare",
      "zone_identifier": "YOUR_ZONE_ID",
      "domain": "example.com",
      "host": "subdomain",
      "token": "YOUR_API_TOKEN",
      "ttl": 120
    }
  ]
}
EOF

# Run the container with mounted volume
docker run -d \
  --name ddns-updater \
  --restart always \
  -p 8000:8000/tcp \
  -v ~/ddns-updater:/updater/data \
  -e LOG_LEVEL=info \
  qmcgaw/ddns-updater:latest
```

### Alternative Method (NOT TESTED)

While the container documentation suggests environment variables can also be used, this method has not been confirmed to work reliably in all situations:

```bash
docker run -d \
  --name ddns-updater \
  --restart always \
  -p 8000:8000/tcp \
  -e SETTINGS_1_PROVIDER=cloudflare \
  -e SETTINGS_1_ZONE_IDENTIFIER=YOUR_ZONE_ID \
  -e SETTINGS_1_DOMAIN=example.com \
  -e SETTINGS_1_HOST=subdomain \
  -e SETTINGS_1_TOKEN=YOUR_API_TOKEN \
  -e SETTINGS_1_TTL=120 \
  -e PERIOD=5m \
  -e LOG_LEVEL=info \
  qmcgaw/ddns-updater:latest
```

**Important Notes:**
- Only the config.json file method has been thoroughly tested and confirmed working in all environments
- Replace `YOUR_ZONE_ID` with your actual Cloudflare Zone ID
- Replace `YOUR_API_TOKEN` with your actual Cloudflare API token
- Splitting the domain into `domain` (example.com) and `host` (subdomain) is critical for proper functioning
- The `TTL` parameter is required (120 seconds is a good value for dynamic DNS)
- The mounted volume approach ensures configuration persistence across container restarts

## Step 4: Check Container Status

```bash
# Check if the container is running properly
docker ps | grep ddns-updater

# View initial logs
docker logs ddns-updater
```

## Step 5: Access the Web Dashboard

The web dashboard will be available at `http://YOUR_SERVER_IP:8000`

You can check this from your server to make sure it's responding:

```bash
# Test if the web UI is accessible locally
curl http://localhost:8000
```

## Troubleshooting

### Common Errors and Solutions

#### "Missing X-Auth-Key, X-Auth-Email or Authorization headers" Error (Code 9106)
- This is a Cloudflare authentication error that commonly occurs after ISP modem restarts
- **Quick fix**: Restart the container with `docker restart ddns-updater`
- If restart doesn't work, recreate the container using the recommended config.json approach in Step 3
- This error indicates one of these issues:
  1. Using an incorrect authentication method (Global API Key instead of API Token)
  2. The API Token has expired or been revoked
  3. The domain and host are not correctly split in your configuration
  4. The configuration wasn't properly applied after a network interruption
- **Solution**: 
  ```bash
  # First check logs for detailed error information
  docker logs ddns-updater
  
  # Recreate the container using the config.json method (most reliable)
  docker rm -f ddns-updater
  
  # Create a proper config.json file with split domain/host
  mkdir -p ~/ddns-updater
  cat > ~/ddns-updater/config.json << 'EOF'
  {
    "settings": [
      {
        "provider": "cloudflare",
        "zone_identifier": "YOUR_ZONE_ID",
        "domain": "example.com",
        "host": "subdomain",
        "token": "YOUR_API_TOKEN",
        "ttl": 120
      }
    ]
  }
  EOF
  
  # Run with the mounted config.json
  docker run -d \
    --name ddns-updater \
    --restart always \
    -p 8000:8000/tcp \
    -v ~/ddns-updater:/updater/data \
    -e LOG_LEVEL=debug \
    qmcgaw/ddns-updater:latest
  ```

#### "zone identifier is not set" Error
- Make sure you're using the correct Zone ID from Cloudflare
- The parameter should be named `zone_identifier` (not `zone_id`)

#### "TTL is not set" Error
- Add the `ttl` parameter to your configuration (e.g., `"ttl":120`)

#### "Cannot unmarshal object into Go struct field" Error
- This occurs when using incorrect JSON structure
- Use the approach shown in Step 3

### File Locations

The configuration file is stored in these locations:

- **Host Machine**: `~/ddns-updater/config.json` (expands to `/home/yourusername/ddns-updater/config.json`)
- **Inside Container**: `/updater/data/config.json`

These locations are linked through the Docker volume mapping: `-v ~/ddns-updater:/updater/data`

To edit the configuration:
```bash
# Edit config with nano
nano ~/ddns-updater/config.json

# After editing, restart the container to apply changes
docker restart ddns-updater
```

### Version Compatibility Notes

- Both the latest version and v2.5.0 work correctly with the configuration shown above
- If you encounter issues with the latest version, you can try v2.5.0 which has been extensively tested:
  ```bash
  docker run -d \
    --name ddns-updater \
    --restart always \
    -p 8000:8000/tcp \
    -v ~/ddns-updater:/updater/data \
    qmcgaw/ddns-updater:v2.5.0
  ```

### Debugging Tips

If authentication issues persist:

1. Check container logs with `docker logs ddns-updater` for specific error messages
2. Try using the `-e LOG_LEVEL=debug` option to get more detailed logs
3. Verify that your Cloudflare API token is still valid by testing it with [Cloudflare's API documentation](https://api.cloudflare.com/)
4. Try recreating the API token in Cloudflare's dashboard

Remember that Cloudflare occasionally makes API changes that might require updates to your configuration.

## Conclusion and Key Learnings

Through extensive testing, we've definitively confirmed:

1. **The config.json file approach with mounted volume is VERIFIED WORKING**
2. **Both the latest container version and v2.5.0 work correctly** with the proper configuration
3. **Critical elements for success:**
   - Using API Token authentication (not Global API Key)
   - Splitting the domain into separate "domain" and "host" fields
   - Mounting a persistent volume for the config.json file
   - Setting appropriate TTL values

The web interface provides a convenient way to monitor your DDNS update status, and the detailed logs help with troubleshooting if any issues occur.

## Common Commands for Management

```bash
# Restart the container
docker restart ddns-updater

# Stop the container
docker stop ddns-updater

# Start the container
docker start ddns-updater

# Remove container and recreate it
docker rm -f ddns-updater
# Then run the docker run command from Step 3 again
```

## References

### Official Documentation
- [DDNS-Updater GitHub Repository](https://github.com/qdm12/ddns-updater) - Official project repository with comprehensive documentation
- [Docker Hub - qmcgaw/ddns-updater](https://hub.docker.com/r/qmcgaw/ddns-updater) - Official Docker image
- [DDNS-Updater Configuration Options](https://github.com/qdm12/ddns-updater/blob/master/docs/configuration.md) - All available configuration options

### Cloudflare Resources
- [Cloudflare API Documentation](https://developers.cloudflare.com/api/tokens/) - Documentation for Cloudflare API tokens
- [Cloudflare DNS Documentation](https://developers.cloudflare.com/dns/) - General DNS configuration in Cloudflare
- [Cloudflare Zone ID Location](https://developers.cloudflare.com/fundamentals/setup/find-account-and-zone-ids/) - How to find your Zone ID
