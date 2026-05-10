# Project Overview

Canvas Conundrum is a collaborative multiplayer puzzle game with educational
trivia elements. Players answer trivia questions to earn resources, then work
together to assemble puzzle pieces on a shared canvas. The GitHub repository
is under the owner "MaxThePrisberry" and is called "canvas-conundrum".

# Orientation

`README.md` has the repo layout and development commands. Don't restate them
here; just read it.

# Design

`game-design.md` and `websocket-events.md` are the source of truth. Code is
derived from these specs, not the other way around. Puzzle tiles are
generated at runtime per game; see `game-design.md` § *Asset Delivery (Puzzle
Images)* and `websocket-events.md` § *Asset Delivery (HTTP)* for the contract.
