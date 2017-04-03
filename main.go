package main

import (
	"fmt"
	"github.com/jirwin/quadlek/lib"
)

func main() {
	fmt.Println("hello world.")
	bot := lib.NewBot("xoxb-46810168865-EAf67TkRnHrWDcaNkulDHbdT")

	bot.StartRTM()
	stop := make(chan bool)
	<-stop
}
