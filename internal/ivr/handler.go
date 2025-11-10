package ivr

import (
	"context"
	"time"

	"github.com/CyCoreSystems/ari/v5"
	"github.com/charmbracelet/log"
)

type ChannelHandler func(ctx context.Context, h *ari.ChannelHandle) error

func Start(ctx context.Context, client ari.Client) {
	sub := client.Bus().Subscribe(nil, "StasisStart")
	defer sub.Cancel()

	subCtx, subCancel := context.WithCancel(context.Background())
	defer subCancel()

	for {
		select {
		case <-ctx.Done():
			log.Warn("context cancelled, exiting ...")
			return

		case evt, ok := <-sub.Events():
			if !ok {
				log.Warn("event channel closed")
				return
			}

			if chanHandl, ok := evt.(*ari.StasisStart); ok {
				log.Infof("c Type = %T", chanHandl)
				go callHandl(ctx, subCtx, subCancel, client.Channel().Get(chanHandl.Key(ari.ChannelKey, chanHandl.Channel.ID)), client)

			}
		}
	}
}

func callHandl(mainCtx context.Context, subCtx context.Context, subCancel context.CancelFunc, h *ari.ChannelHandle, client ari.Client) {
	h.Answer()
	time.Sleep(2 * time.Second)
	defer h.Hangup()

	mainCtx, cancel := context.WithCancel(mainCtx)
	defer cancel()

	log.Info("Runnign app", "Channel", h.ID())

	//Welcomming message
	go welcomeMessage(mainCtx, subCtx, h)

	// actions := map[string]func(){
	// 	"1":       func() { RecordingRequest(mainCtx, h) },
	// 	"2":       func() { StopCall(mainCtx, h) },
	// 	"default": func() { DoNothing(mainCtx, h) },
	// }
	DTMFHandl(mainCtx, subCancel, client, h, firstRecord()) //First record

	end := h.Subscribe(ari.Events.StasisEnd)
	defer end.Cancel()

	//End the app when the channel goes away
	go func() {
		<-end.Events()
		cancel()
	}()

}

func StopCall(ctx context.Context, h *ari.ChannelHandle) error {
	err := playSound(ctx, h, "sound:vm-goodbye")
	log.Info("Stopping call", "Channel", h.ID())
	h.Hangup()
	return err
}
func DoNothing(ctx context.Context, h *ari.ChannelHandle) error {

	log.Info("Doing nothing for Channel", "Channel", h.ID())
	return nil
}
