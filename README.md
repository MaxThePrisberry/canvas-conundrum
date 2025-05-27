# The Canvas Conundrum

# Setup

## Install Golang
See instructions in the [Golang Download and Install page](https://go.dev/doc/install). Get at least Golang 1.21 or higher.

## Install `pre-commit`
In Ubuntu, run:
```
sudo apt update; sudo apt install pre-commit -y
cd canvas-conundrum; pre-commit install
```

Install Golang packages for pre-commit hooks:
```
go install golang.org/x/tools/cmd/goimports@latest
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

# Attribution
Trivia provided by the [Open Trivia Database](https://opentdb.com/) under the [Creative Commons Attribution-ShareAlike 4.0 International License](https://creativecommons.org/licenses/by-sa/4.0/) with no changes made. Thank you!
