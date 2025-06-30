go test -v -count=1 -race ./client/game/state/ ./tests/ -coverprofile cover.out
go tool cover -html=cover.out -o cover.html
