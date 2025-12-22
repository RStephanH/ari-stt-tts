package ivr

import (
	"context"
	"time"

	"github.com/CyCoreSystems/ari/v5"
	"github.com/charmbracelet/log"
)

func DTMFHandl(mainCtx context.Context,
	sound string, client ari.Client,
	ch *ari.ChannelHandle,
	actions map[string]ChannelHandler,
	listDigOpt []string,
) {

	sub := client.Bus().Subscribe(nil, "RecordingFinished")
	defer sub.Cancel()
	//
	// }()

	for {
		select {

		case <-mainCtx.Done():
			return
		default:

			if res, er := promptSound(mainCtx, ch, sound, listDigOpt, 3); er == nil {

				if action, ok := actions[res.DTMF]; ok {
					if err := action(mainCtx, ch); err != nil {
						log.Error("Error executing action for DTMF digit", "Digit", res.DTMF, "Error", err)
					}
					if res.DTMF == "1" {
						for evts := range sub.Events() {
							if evt, ok := evts.(*ari.RecordingFinished); ok {
								log.Infof("Recording finished: %s", evt.Recording.Name)
								log.Info("Should switch on another function")
								return
							}

						}
					} else {
						log.Info("Action terminated")
						return
					}
				} else if res.DTMF == "#" {
					actions["default"](mainCtx, ch)

				} else if res.DTMF == "" {
					time.Sleep(100 * time.Millisecond)
					continue

				} else {
					log.Warn("No action defined for this DTMF digit", "Digit", res.DTMF)
				}

				// }
			} else {
				log.Error("Error during prompt sound", "Error", er)
				return
			}
		}

	}
}
