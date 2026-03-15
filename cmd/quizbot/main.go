//go:build !solution

package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/ilkaln/quiz-bot/internal/quiz"
	"github.com/ilkaln/quiz-bot/internal/telegram"
)

func main() {
	token := flag.String("token", "", "Telegram Bot API Token")
	username := flag.String("bot-username", "", "Bot username without @")

	flag.Parse()

	if *token == "" {
		fmt.Printf("Skiped token.\n")
		os.Exit(1)
	}

	if *username == "" {
		fmt.Printf("Skiped bot-username.\n")
		os.Exit(1)
	}

	fmt.Printf("Starting bot @%s with token %s\n", *username, *token)

	clientHTTPS := telegram.NewHTTPClient(*token)

	engine := quiz.NewEngine()

	bot := telegram.NewBot(clientHTTPS, engine, *username)

	bot.Run()
}
