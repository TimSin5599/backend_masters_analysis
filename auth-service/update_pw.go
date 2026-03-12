package main

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	password := "password"
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal(err)
	}

	conn, err := pgx.Connect(context.Background(), "postgres://user:password@localhost:5433/fqw_db")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close(context.Background())

	_, err = conn.Exec(context.Background(), "UPDATE users SET password = $1 WHERE email = 'admin@flatlogic.com'", string(hash))
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Password updated successfully!")
}
