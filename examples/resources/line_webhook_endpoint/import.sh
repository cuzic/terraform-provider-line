# The webhook endpoint is a channel-level singleton, so any non-empty
# string works as the import ID.
terraform import line_webhook_endpoint.this webhook_endpoint
