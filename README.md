# Game Student

This project implements the API of the game student website in go  

## Getting Started

Below are a few commands you can use to test the project. Make sure 
to read Makefile for all the commands.

### Building the project

```bash
make build
```

This will created executables on dist folder: 

api is the server
migration runs the migrations for the server

### Running dependencies

```bash
make deps
```

This will start the database container

### Running migrations

```bash
make migrate
```

This will configure the database

### Running the server

```bash
make run
```

This will start the API