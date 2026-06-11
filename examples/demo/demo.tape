# demo.tape — sqlh animated demo for README
# Generated for issue #25

Output docs/demo.gif

# Terminal dimensions and style
Set Width 960
Set Height 520
Set FontSize 13
Set FontFamily "JetBrains Mono"
Set Theme "Dracula"
Set Padding 20

# Typing speed
Set TypingSpeed 60ms

# 1. Show the project root
Type "cd examples/demo"
Enter
Sleep 300ms

# 2. Show source code (struct definition — the USP)
Type "head -n 35 main.go"
Enter
Sleep 5s

# 3. Clear and run the demo
Ctrl+C
Sleep 200ms

Type "go run ."
Enter
Sleep 5s

# 4. Pause on final output
Sleep 2s
