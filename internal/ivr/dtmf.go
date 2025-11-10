package ivr

import (
	"context"

	"github.com/CyCoreSystems/ari/v5"
	"github.com/charmbracelet/log"
)

func DTMFHandl(mainCtx context.Context, subCancel context.CancelFunc, client ari.Client, ch *ari.ChannelHandle, actions map[string]ChannelHandler) {
	// TODO add functionality to handle DTMF events with functions as parameters
	sub := client.Bus().Subscribe(nil, "ChannelDtmfReceived")
	defer sub.Cancel()

	mainCtx, cancel := context.WithCancel(mainCtx)

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
						action(mainCtx, ch)
					} else if ev.Digit == "#" {
						actions["default"](mainCtx, ch)

					} else {
						log.Warn("No action defined for this DTMF digit", "Digit", ev.Digit)
					}

				}
			}

		case <-mainCtx.Done():
			cancel()

		}
	}
}
