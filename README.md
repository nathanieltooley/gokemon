<div align="center">
  <img src="https://github.com/user-attachments/assets/11738814-cb8a-4349-884d-a66fbe5ddfca" width="50%"/>
</div>

# Gokemon
Gokemon is a CLI Pokemon battle simulator written in Go. Gokemon attempts to be a fairly accurate, if feature-lacking, simulation of newer-gen Pokemon battle mechanics.
Gokemon will not feature previous generations' mechanics, abilities, stats, or bugs the way a battle simulator like Pokemon Showdown does.

> [!WARNING]
> This project is still very much under construction! Gokemon currently contains the Pokemon data of Gen 1 Pokemon and <50% of abilities introduced in Gen 3.
> The goal for now is to get all Pokemon and Pokemon abilites from Gens 1-3 added into the game (though the mechanics would be from Gen 8). If there are any features you wish to see added, feel free to open
> an issue or pull request.

## Installation
The easiest way to install is to download the latest pre-compiled binary executable for your system from the [releases tab](https://github.com/nathanieltooley/gokemon/releases). 

> [!NOTE]
> The executable needs to be run inside of a terminal / command prompt. Double-clicking or otherwise directly running the executable will output nothing.

### Building from source

The other way is to build / run the code directly on your computer using Go.

First you must install [Go](https://go.dev/). Follow your platform-specific installation instructions. 

Next clone the repo into a folder of your chosing:
```
git clone https://github.com/nathanieltooley/gokemon.git
```
Inside the cloned repo you can build Gokemon using the following command:
```
go build main.go
```
to change the name of the executable, use the -o flag:
```
go build -o gokemon main.go
```
or run the game without building an executable:
```
go run main.go
```
