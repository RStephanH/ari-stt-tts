package ivr

import (
	"context"
	"time"

	"ari/internal/ariutil"

	"github.com/CyCoreSystems/ari/v5"
	"github.com/charmbracelet/log"
)

type ChannelHandler func(ctx context.Context, h *ari.ChannelHandle) error

// answerMediaSettleDelay compensates for a brief media-negotiation gap right
// after Answer(). Historically needed due to ASTERISK-24229 (fixed upstream
// in 12.6+) -- worth re-testing (and possibly removing) once you can listen
// for clipping on this Asterisk version.
const answerMediaSettleDelay = 300 * time.Millisecond

func Start(ctx context.Context, client ari.Client, ariConfig ariutil.Config) {
	sub := client.Bus().Subscribe(nil, "StasisStart")
	defer sub.Cancel()

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
				log.Infof("Events StasisStart Type = %T", chanHandl)
				go callHandl(
					ctx,
					client.Channel().Get(chanHandl.Key(ari.ChannelKey, chanHandl.Channel.ID)),
					ariConfig,
				)
			}
		}
	}
}

func callHandl(mainCtx context.Context, h *ari.ChannelHandle, ariConfig ariutil.Config) {
	if err := h.Answer(); err != nil {
		log.Error("Failed to answer channel", "channel", h.ID(), "error", err)
		return
	}
	time.Sleep(answerMediaSettleDelay)
	defer h.Hangup()

	mainCtx, cancel := context.WithCancel(mainCtx)
	defer cancel()

	log.Info("Running app", "channel", h.ID())

	end := h.Subscribe(ari.Events.StasisEnd)
	defer end.Cancel()

	go func() {
		<-end.Events()
		cancel()
	}()

	session := newCallSession(h, ariConfig)

	if err := DTMFHandl(mainCtx, "sound:welcome-ari", h, firstRecord(session), []string{"1", "0", "#"}); err != nil {
		log.Error("First record step failed", "channel", h.ID(), "error", err)
		return
	}

	if err := DTMFHandl(mainCtx, "sound:after_recording", h, secondRecord(session), []string{"1", "2", "3", "0", "#"}); err != nil {
		log.Error("Second record step failed", "channel", h.ID(), "error", err)
		return
	}

	for {
		select {
		case <-mainCtx.Done():
			log.Info("Main context done, exiting DTMF handler loop", "channel", h.ID())
			return
		default:
			if err := DTMFHandl(mainCtx, "sound:after_responding", h, thirdRecord(session), []string{"1", "2", "3", "4", "0", "#"}); err != nil {
				log.Error("Third record step failed", "channel", h.ID(), "error", err)
				return
			}
		}
	}
}

func StopCall(ctx context.Context, h *ari.ChannelHandle) error {
	err := PlaySound(ctx, h, "sound:ari_goodbye")
	log.Info("Stopping call", "channel", h.ID())
	h.Hangup()
	return err
}

func DoNothing(ctx context.Context, h *ari.ChannelHandle) error {
	log.Info("Doing nothing for channel", "channel", h.ID())
	return nil
}
