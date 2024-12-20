# VHS documentation
#
Set Shell "bash"

Set FontSize 13                 # Set the font size of the terminal
Set FontFamily "Menlo"          # Set the font family of the terminal
Set Height 880                  # Set the height of the terminal, in pixels
Set Width 864                   # Set the width of the terminal, in pixels
Set Padding 1                   # Set the padding of the terminal
Set BorderRadius 4              # Set terminal border radius, in pixels.

Output vhs.gif

Hide
Type "go run ../cmd/tui/main.go"
Enter
Show

Sleep 20s

Ctrl+C
