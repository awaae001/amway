package main

import (
	"amway/bot"
	"amway/utils"
)

func main() {
	utils.InitDB()
	bot.Start()
}
