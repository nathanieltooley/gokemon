<div align="center">
  <img src="https://github.com/user-attachments/assets/11738814-cb8a-4349-884d-a66fbe5ddfca" width="50%"/>
</div>

# Gokemon
Gokemon is a CLI Pokemon battle simulator written in Go. Gokemon attempts to be a fairly accurate, if feature-lacking, simulation of Pokemon singles battles. Singleplayer against an AI
and LAN multiplayer modes are available. Gokemon will not feature previous generations' mechanics, abilities, stats, or bugs the way a battle simulator like Pokemon Showdown does.
All mechanics and Pokemon are based off of their latest appearances.

> [!WARNING]
> This project is still very much under construction! Gokemon currently contains the Pokemon data of Gen 1 - 3 Pokemon and ~99% of abilities introduced in Gen 3.
> While move data up to Gen 9 is included, most moves are not fully implemented (Like Protect blocking damage, Substitution, etc.)
> The next major plan is to add in-battle items. Double battles are up in the air. If there are any features you wish to see added, feel free to open an issue or pull request.

## Installation
The easiest way to install is to download the latest pre-compiled binary executable for your system from the [releases tab](https://github.com/nathanieltooley/gokemon/releases). 

> [!NOTE]
> Windows will probably flag this as a virus and try to delete the file or prevent it's execution. I would recommend letting it through the antivirus rather than temporarily disabling it
> when running the program. I don't know if theres a realistic way for me to fix this so you'll just have to trust me :)

### Building from source

The other way is to build / run the code directly on your computer using Go.

First you must install [Go](https://go.dev/). Follow your platform-specific installation instructions. 

Next clone the repo into a folder of your chosing:
```
git clone https://github.com/nathanieltooley/gokemon.git
```
Inside the cloned repo you can build Gokemon using the following command:
```
go build -o gokemon ./poketerm
```
And then execute:
```
./poketerm
```
or run the game without building an executable:
```
go run ./poketerm
```
