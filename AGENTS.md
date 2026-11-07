# GoRex

## Build and install

```sh
go build ./...                    # check it compiles
go test ./...                     # the app tests start a real session server
scripts/install.sh --no-install   # build and sign build/darwin-arm64/GoRex.app
scripts/install.sh                # build, sign and copy to /Applications
```

`scripts/install.sh` signs with "GoRex Local", a self-signed certificate in
the login keychain, so macOS keeps GoRex's folder permissions across builds.
Don't build with `mygo build -sign`: it turns on the hardened runtime,
which refuses libghostty-vt.dylib, and no session opens.

The install stops while GoRex runs. Leave quitting it to the user: agents
often run inside a GoRex session, which the quit can cut off. The session
server (`GoRex -server`) holds every session; never kill it.
