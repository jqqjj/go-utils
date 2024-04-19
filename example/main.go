package main

import (
	"fmt"
	"github.com/jqqjj/go-utils"
	"log"
)

type eventType int
type EventType = utils.EnumInt[eventType]

type eventType2 int
type EventType2 = utils.EnumInt[eventType2]

var (
	ClickEvent = utils.NewEnumInt[eventType](0)
	MouseEvent = utils.NewEnumInt[eventType](1)

	ClickEvent2 = utils.NewEnumInt[eventType2](2)
)

func main() {
	parseInt, err := utils.EnumIntParse[EventType](0)
	if err != nil {
		log.Fatalln(err)
		return
	}

	fmt.Println(parseInt.Setted(), parseInt == ClickEvent)
}
