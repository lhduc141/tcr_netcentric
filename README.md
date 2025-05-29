=======================================
Technical Documentation - Tower Clash Royale (TCR) Server
=======================================

1. Overview

This project implements a two-player Tower Clash Royale (TCR) game server based on TCP connections. The game features turn-based combat between two players, each controlling towers and troops. The server handles authentication, game logic, combat resolution, and player progression (EXP).

---------------------------------------

2. Architecture

- Server: TCP server listening on port 9000, managing two concurrent players per game session.
- Client: Simple TCP client CLI for command input and server message display.
- Data Storage: Player credentials, EXP, and game entity specs (troops, towers) stored in JSON files on disk.
- Concurrency: Uses Go goroutines and mutex locks for safe concurrent player management.

---------------------------------------

3. Components

3.1 Server (server.go)

- Listens for client connections and authenticates users via JSON file (players.json).
- Manages player sessions and game state.
- Implements game loop with turn-based command processing.
- Loads tower and troop specifications from JSON.
- Manages mana regeneration and troop summoning.
- Resolves combat between troops and towers lane-by-lane.
- Broadcasts game events to all connected clients.
- Updates player EXP on game completion.
- Terminates game and disconnects players on win condition.

3.2 Utilities (utils package)

- Battle logic (battle.go)
  - CalculateDamage: Computes damage including critical hits.
  - HealTower: Heals the tower with the lowest HP.

- Data loaders (loader.go)
  - Load troop specs from troop_specs.json.
  - Load tower specs from tower_specs.json.

3.3 Models (models package)

- Troop
  - Attributes: Name, HP, ATK, DEF, Mana, EXP, Special, Lane, TimeToTarget, Owner, Alive.

- Tower
  - Attributes: Name, HP, ATK, DEF, CRIT, EXPValue.

3.4 Client (client.go)

- Command line TCP client for connecting to the server.
- Reads user input from stdin and sends to server.
- Listens for and displays server messages asynchronously.

---------------------------------------

4. Game Flow

4.1 Connection and Authentication

- Players connect to server on TCP port 9000.
- Server prompts for username and password.
- Authenticates credentials against players.json.
- Once 2 players connected and authenticated, game starts.

4.2 Game Initialization

- Loads tower specs (GuardTower and KingTower).
- Assigns each player 3 towers.
- Initializes player mana to 5.

4.3 Turn-Based Play Loop

- Players alternate turns.
- Each turn:
  - Server prompts current player for commands (summon, end).
  - Players summon troops if sufficient mana.
- After both players submit commands:
  - Resolve combat lane by lane.
  - Troops fight troop vs troop.
  - Surviving troops attack enemy towers considering DEF and CRIT.
  - Update tower HP and DEF accordingly.
  - Remove destroyed troops.
  - Announce events via broadcast.
  - Mana regenerates by 2 per player (max 10).
- Repeat until win condition met.

4.4 Win Condition

- A player wins by destroying the opponent's King Tower.
- On win:
  - Update EXP (+30 to winner, +10 to loser).
  - Broadcast win message.
  - Disconnect players and terminate server.

---------------------------------------

5. Data Storage

- Player credentials and EXP: stored in data/players.json.
- Troop specs: stored in data/troop_specs.json.
- Tower specs: stored in data/tower_specs.json.
- Player EXP is updated live after each game.

---------------------------------------

6. Command Protocol

Commands sent from client to server are text-based:

- summon <TroopName> <guard|king> <Number>
  - Summons a troop to a lane.
  - Example: summon Archer guard 1

- end
  - Ends player's turn.

Server sends prompts, feedback, and game event messages as plain text.

---------------------------------------

7. Known Limitations and Scope

- Game mode is turn-based only (no real-time continuous mode).
- Mana regeneration is per turn (+2), not continuous over time.
- EXP and level system basic: level derived from EXP/100, no stat boosts implemented.
- No persistent troop/tower leveling or stat upgrades saved.
- Client is a simple CLI, no GUI.
- Only handles 2-player games; no matchmaking or multiple concurrent games.
- Game ends immediately when King Tower is destroyed; no timed match mode.
- CRIT chance used only for tower damage, not troop vs troop combat.

---------------------------------------

8. Future Enhancements (Out of Current Scope)

- Implement real-time game mode with continuous combat and timer.
- Add mana regen per second and mana cost system improvements.
- Add troop and tower leveling with stat scaling on EXP.
- Implement UI client with graphical interface.
- Support multiple concurrent games and matchmaking.
- Save player progress and stats more comprehensively.
- Add logging and monitoring for better maintainability.

---------------------------------------

9. Setup and Running

1. Place JSON config files (players.json, troop_specs.json, tower_specs.json) in data/ folder.
2. Run server: go run server.go
3. Run clients: go run client.go
4. Follow command prompts to play.

---------------------------------------

10. Code Structure Summary

tcr_netcentric/
├── data/
│   ├── players.json
│   ├── troop_specs.json
│   └── tower_specs.json
├── models/
│   ├── troops.go
│   └── tower.go
├── utils/
│   ├── battle.go
│   └── loader.go
├── server.go
└── client.go

=======================================
