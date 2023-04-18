# YatzCLI

YatzCLI is a turn-based multiplayer command-line game inspired by the classic dice game Yahtzee. Designed with developers in mind, it's the perfect way to kill time while waiting for builds to finish or during those dull training sessions. With a simple interface and quick matches, you can effortlessly engage in some friendly competition with your peers.

## How it works

YatzCLI is built using Go and leverages the standard library's networking capabilities to connect clients to a central server. The game follows a turn-based structure, allowing players to roll dice, reroll, and choose scoring categories while ensuring smooth and seamless gameplay. The server handles all the game logic and state, broadcasting updates to the clients as the game progresses.

## How to run

Clone the repository:

```bash
git clone https://github.com/edge2992/yatzcli.git
```

Change to the project directory:

```bash
cd yatzcli
```

Build the server and client executables:

```bash
go build -o server server/main.go
go build -o client client/main.go
```

Run the server in a separate terminal:

```bash
./server
```

Run the client in another terminal (one for each player):

```bash
./client
```

Once connected, the game will start automatically when two or more players have joined. Follow the on-screen prompts to play your turn, reroll dice, and choose categories for scoring. Enjoy the game and happy coding!

## Game flow

The game follows a typical turn-based flow:

1. Players connect to the server, joining a game room.
2. When two or more players are in the room, the game starts automatically.
3. The server shares the current turn with all participants.
4. The server shares the rolled dice with all participants.
5. The server notifies the current player and asks if they want to reroll the dice.
6. The player can reroll the dice up to two times, with the server sharing the updated dice after each reroll.
7. The player chooses a scoring category for the rolled dice.
8. The server updates the scorecard and moves to the next player's turn.

The game continues until all players have completed their turns and filled their scorecards. At the end of the game, the server calculates the final scores and declares the winner.
