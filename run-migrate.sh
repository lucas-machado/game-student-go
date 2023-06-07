#!/bin/bash

export DB_CONN="user=ps_user password=ps_password dbname=backend sslmode=disable host=localhost"

./dist/migrate
