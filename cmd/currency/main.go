package main

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Overload()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	currencyKey := os.Getenv("CURRENCY_API_KEY")
	fmt.Println("Currency API key: " + currencyKey)
}
