package ivr

import (
	"context"

	"github.com/CyCoreSystems/ari/v5"
	"github.com/charmbracelet/log"
)

func DTMFHandl(mainCtx context.Context, mainCancel context.CancelFunc, subCancel context.CancelFunc, client ari.Client, ch *ari.ChannelHandle, actions map[string]ChannelHandler) {
	sub := client.Bus().Subscribe(nil, "ChannelDtmfReceived", "RecordingFinished")
	defer sub.Cancel()

	for {
		select {

		case e := <-sub.Events():
			if ev, ok := e.(*ari.ChannelDtmfReceived); ok {

				if ev.Channel.ID == ch.ID() {

					if action, ok := actions[ev.Digit]; ok {
						go func() {
							subCancel()
							log.Info("Stop any Playback message and should record now")

						}()

						if err := action(mainCtx, ch); err != nil {
							log.Error("Error executing action for DTMF digit", "Digit", ev.Digit, "Error", err)
						}
						for evts := range sub.Events() {
							if evt, ok := evts.(*ari.RecordingFinished); ok {
								log.Infof("Recording finished: %s", evt.Recording.Name)
								log.Info("Should switch on another function")
								return
							}
						}
					} else if ev.Digit == "#" {
						actions["default"](mainCtx, ch)

					} else {
						log.Warn("No action defined for this DTMF digit", "Digit", ev.Digit)
					}

				}
			}

		case <-mainCtx.Done():
			return

		}
	}
}
