<div align="center">
  <img alt="The Gokemon Logo: a pokeball stylized like the Golang Gopher" src="https://github.com/user-attachments/assets/11738814-cb8a-4349-884d-a66fbe5ddfca" width="50%"/>
</div>

# Gokemon
Gokemon is a CLI Pokemon battle simulator written in Go. Gokemon attempts to be a fairly accurate, if feature-lacking, simulation of Pokemon singles battles. Singleplayer against an AI
and LAN multiplayer modes are available. All mechanics and Pokemon are based off of their latest appearances (as of 2024).

> [!WARNING]
> This project is still very much under construction! Gokemon currently contains the Pokemon data of Gen 1 - 3 Pokemon and ~99% of abilities introduced in Gen 3.
> While move data up to Gen 9 is included, most moves' secondary effects are not implemented
> If there are any features you wish to see added, feel free to open an issue or pull request.


<div align="center">
  <img width="75%" alt="A view of the battle screen. Waiting for the player to select an action." src="https://github.com/user-attachments/assets/a10a2a70-2db5-4597-a995-af4fdd5d9ea3" />
</div>

## Installation
The easiest way to install is to download the latest pre-compiled binary executable for your system from the [releases tab](https://github.com/nathanieltooley/gokemon/releases). 

> [!NOTE]
> Windows may or may not flag this as a virus / malware so Windows Antivirus may have to be disabled during installation and explicitly allowed.
> Building / Running from source should avoid this issue. If you know why this happens and how to fix it do feel free to open an issue.

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
