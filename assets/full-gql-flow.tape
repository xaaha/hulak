Output assets/full-gql-flow.gif
Set Shell "bash"
Set FontSize 18
Set Width 1920
Set Height 1080
Set Padding 30
Set TypingSpeed 80ms
Set Theme "Dracula"

Type "mkdir -p demo"
Enter
Sleep 1s

Type "cd demo"
Enter
Sleep 1s

Type "hulak init staging"
Enter
Sleep 1s

Type "cat > anilist.yml"
Enter
Type "# description: fetch real gql schema for countries"
Enter
Type "---"
Enter
Type "kind: GraphQL"
Enter
Type "url: https://graphql.anilist.co"
Enter
Type "headers:"
Enter
Type "  Content-Type: application/json"
Enter
Ctrl+d
Sleep 1s

Type "cat > countries.yml"
Enter
Type "# description: fetch real gql schema for countries"
Enter
Type "---"
Enter
Type "kind: GraphQL"
Enter
Type "url: https://countries.trevorblades.com/"
Enter
Type "headers:"
Enter
Type "  Content-Type: application/json"
Enter
Ctrl+d
Sleep 1s

Type "hulak gql ."
Enter
Sleep 1s

Type "country"
Enter
Enter
Sleep 1s

Space
Type "BR"
Sleep 500ms

Ctrl+O
Sleep 1s

Tab
Sleep 1s

Tab
Sleep 1s

Tab
Sleep 1s
Ctrl+S
Sleep 2s
Escape
Sleep 2s
Ctrl+X
Sleep 2s
Escape
