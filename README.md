# yuno-go
Attempt at porting Yuno Gasai 2 bot to golang
Quickstart guide
# 1. Clone & enter
git clone https://github.com/blubskye/yuno-go.git
cd yuno-go

# 2. Edit config (or let it auto-generate)
nano config.toml
# → Put your token and owner ID

# 3. Run (that's it)
go run .

# Or build a single binary:
go build
./yuno-go
