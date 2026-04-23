Output casette.gif
Set Shell zsh
Set FontSize 32
Set Width 1200
Set Height 600

Require gotpm

Sleep 500ms
Type "gotpm init foo -v" Sleep 500ms Enter
Sleep 3s

Hide
Type "clear" Enter
Show

Type "gotpm install foo -e" Sleep 500ms 
Enter
Sleep 3s

Hide
Type "clear" Enter
Show

Type "gotpm list" Sleep 500ms 
Enter
Sleep 3s

Hide
Type "clear" Enter
Show

Type "gotpm uninstall foo --all" Sleep 500ms 
Enter
Sleep 3s
