You have access to /home/2qar/gaming/git/UndertaleModTool/UndertaleModCli/bin/Release/net8.0/UndertaleModCli. Use it to dump the code for this file: /home/2qar/gaming/old-hero-siege/bin/data.win

The game you are dumping makes various HTTP calls, but the backend no longer exists.

I already know how to modify the data.win to point at a new backend. But I need you to write the functionality.

Let's write it in Go in this repository, following a few constraints:
1. Only use the built-in golang packages wherever possible. Exceptions for SQLite and UUID if needed
2. SQLite is our database.
3. Each of the endpoints is implemented as a distinct handler.
4. Components should be decoupled - e.g. AccountManager, SessionManager, etc.
5. Functional tests must be written against the e2e app. They should be able to spin up an independent copy of the server with a fresh database in a temporary directory, and make real requests against the server as if it were the game. You can ask for my inputs on this
6. The whole repo should be AI-native, but we want minimal steering. Just the bare-minimum for an agent to work on this. We will have enforced gofmt on pre-commit, linter, etc. to ensure the code is high-quality DETERMINISTICALLY.
7. Any agent documentation should not re-state what could just be learned by looking at the code itself
