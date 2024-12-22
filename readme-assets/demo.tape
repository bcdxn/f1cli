# VHS documentation
#
Set Shell "bash"

Set FontSize 12                 # Set the font size of the terminal
Set FontFamily "Menlo"          # Set the font family of the terminal
Set Height 960                  # Set the height of the terminal, in pixels
Set Width 880                   # Set the width of the terminal, in pixels
Set Padding 5                   # Set the padding of the terminal, in pixels
Set BorderRadius 10              # Set terminal border radius, in pixels.
Set WindowBar Colorful
Set Margin 30
Set MarginFill "#3B4252"

Output demo.gif

Type@300ms "f1"
Sleep 600ms
Hide
Ctrl+C
Type "go run ../cmd/tui/main.go"
Enter
Type "clear"
Sleep .1
Show

Sleep 15s

Ctrl+C
