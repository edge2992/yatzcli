# YatzCLI

Set up the project structure:
Create a new directory for your project and initialize it as a Go module. You can do this by running go mod init <module-name> in your project directory.

Define the data structures:
In a new file, yahtzee.go, define the data structures for the game, such as Player, Dice, and ScoreCard. You'll also need to define the various scoring categories for Yahtzee, such as ThreeOfAKind, FourOfAKind, FullHouse, etc.

Implement the game mechanics:
Create functions for rolling the dice, scoring the dice, and determining if a roll can be scored in a particular category. You'll also need to implement game state management, including tracking players' turns and scores.

Create the CLI:
In another file, main.go, create a command-line interface to interact with the game. You can use the fmt package to handle input and output. You'll need to prompt the user to roll the dice, hold or re-roll dice, and choose a scoring category. Also, implement a loop for taking turns and managing the game state.

Add the automatic matching function:
To add an automatic matching function, you can implement a simple AI player that chooses the best scoring category for a given roll, based on some heuristics. Alternatively, you could use a more advanced approach, such as a Monte Carlo tree search or a neural network.
