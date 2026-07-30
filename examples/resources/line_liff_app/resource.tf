resource "line_liff_app" "this" {
  view_type  = "full"
  view_url   = "https://example.com/liff-app"
  bot_prompt = "none"
  scope      = ["profile", "chat_message.write"]
}
