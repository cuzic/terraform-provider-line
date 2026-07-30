terraform {
  required_providers {
    line = {
      source = "cuzic/line"
    }
  }
}

provider "line" {
  # channel_access_token can also be supplied via the LINE_CHANNEL_ACCESS_TOKEN
  # environment variable, which is the recommended way to avoid writing it
  # into configuration or committing it to version control.
  channel_access_token = var.line_channel_access_token
}
