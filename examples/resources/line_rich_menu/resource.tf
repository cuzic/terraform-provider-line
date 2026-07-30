resource "line_rich_menu" "this" {
  name          = "nice rich menu"
  chat_bar_text = "Menu"
  selected      = false
  image_path    = "${path.module}/rich-menu.png"

  size = {
    width  = 2500
    height = 1686
  }

  areas = [
    {
      bounds = {
        x      = 0
        y      = 0
        width  = 2500
        height = 1686
      }
      action = {
        type  = "postback"
        data  = "action=buy&itemid=123"
        label = null
        text  = null
        uri   = null
      }
    }
  ]
}
