package test

import (
	"context"
	"github.com/jqqjj/go-utils"
	"testing"
	"time"
)

type attachmentType string
type AttachmentType = utils.EnumString[attachmentType]

var (
	AttachmentTypePhoto = utils.NewEnumString[attachmentType]("1")
	AttachmentTypeVideo = utils.NewEnumString[attachmentType]("2")
)

func TestPubSub(t *testing.T) {
	e := utils.NewPubSub[AttachmentType, []byte]()

	ctxPhoto, cancelPhoto := context.WithCancel(context.Background())
	ctxVideo, cancelVideo := context.WithCancel(context.Background())
	defer cancelPhoto()
	defer cancelVideo()

	photo := make(chan []byte)
	video := make(chan []byte)

	e.Publish(AttachmentTypePhoto, []byte("photo"))
	e.Publish(AttachmentTypeVideo, []byte("video"))

	e.Subscribe(ctxPhoto, AttachmentTypePhoto, photo)
	e.Subscribe(ctxVideo, AttachmentTypeVideo, video)

	e.Publish(AttachmentTypePhoto, []byte("1"))
	e.Publish(AttachmentTypePhoto, []byte("2"))
	e.Publish(AttachmentTypeVideo, []byte("1"))
	e.Publish(AttachmentTypeVideo, []byte("2"))
	e.Publish(AttachmentTypeVideo, []byte("3"))
	e.Publish(AttachmentTypePhoto, []byte("1"))

	select {
	case v := <-video:
		if string(v) != "1" {
			t.Error("0")
			return
		}
	default:
		t.Error("0")
		return
	}

	if e.SubscriberCountOfTopic(AttachmentTypeVideo) != 1 {
		t.Error("1", e.SubscriberCountOfTopic(AttachmentTypeVideo))
		return
	}

	cancelVideo()

	time.Sleep(time.Second)

	if e.SubscriberCountOfTopic(AttachmentTypeVideo) != 0 {
		t.Error("2", e.SubscriberCountOfTopic(AttachmentTypeVideo))
		return
	}

}
