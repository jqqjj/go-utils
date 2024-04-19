package test

import (
	"encoding/json"
	"github.com/jqqjj/go-utils"
	"testing"
)

type evenType int
type EventType = utils.EnumInt[evenType]

type event2Type string
type Event2Type = utils.EnumString[event2Type]

type event3Type string
type Event3Type = utils.EnumString[event3Type]

var (
	Default3    = utils.NewEnumString[event3Type]("")
	ClickEvent3 = utils.NewEnumString[event3Type]("click")

	Default2    = utils.NewEnumString[event2Type]("")
	ClickEvent2 = utils.NewEnumString[event2Type]("click2")

	ClickEvent = utils.NewEnumInt[evenType](0)
	MouseEvent = utils.NewEnumInt[evenType](1)
	Default    = utils.NewEnumInt[evenType](2)
)

func TestEnum(t *testing.T) {
	type E struct {
		Event  Event2Type `json:"event"`
		Event1 Event2Type `json:"event1"`
		Event2 Event2Type `json:"event2"`
		Event3 Event2Type `json:"event3"`

		Event4 EventType `json:"event4"`
		Event5 EventType `json:"event5"`
		Event6 EventType `json:"event6"`
	}

	var e1 E
	str := `{"event":"click2","event1":"","event2":"click","event3":null,"event4":null,"event5":3,"event6":0}`
	if err := json.Unmarshal([]byte(str), &e1); err != nil {
		t.Error(err)
		return
	}

	if !e1.Event.IsEmpty() || !e1.Event1.IsEmpty() || e1.Event2.IsEmpty() || e1.Event3.IsEmpty() {
		t.Fail()
		return
	}
	if e1.Event4.IsEmpty() || e1.Event5.IsEmpty() || !e1.Event6.IsEmpty() {
		t.Fail()
		return
	}

	parseInt, err := utils.EnumStringParse[Event3Type]("")
	if err != nil {
		t.Error(err)
		return
	}

	if !parseInt.IsEmpty() || parseInt != Default3 || parseInt == ClickEvent3 {
		t.Fail()
		return
	}
}
