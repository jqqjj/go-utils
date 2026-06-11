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
	event3Types = utils.NewEnumStringType[event3Type]()
	Default3    = event3Types.Add("")
	ClickEvent3 = event3Types.Add("click")

	event2Types = utils.NewEnumStringType[event2Type]()
	Default2    = event2Types.Add("")
	ClickEvent2 = event2Types.Add("click2")

	eventTypes = utils.NewEnumIntType[evenType]()
	ClickEvent = eventTypes.Add(0)
	MouseEvent = eventTypes.Add(1)
	Default    = eventTypes.Add(2)
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
	str := `{"event":"click2","event1":"","event2":null,"event3":null,"event4":null,"event5":1,"event6":0}`
	if err := json.Unmarshal([]byte(str), &e1); err != nil {
		t.Error(err)
		return
	}

	if e1.Event != ClickEvent2 || e1.Event1 != Default2 || e1.Event2.IsSet() || e1.Event3.IsSet() {
		t.Fail()
		return
	}
	if e1.Event4.IsSet() || e1.Event5 != MouseEvent || e1.Event6 != ClickEvent {
		t.Fail()
		return
	}

	parseInt, err := event3Types.Parse("")
	if err != nil {
		t.Error(err)
		return
	}

	if parseInt != Default3 || parseInt == ClickEvent3 {
		t.Fail()
		return
	}
}
