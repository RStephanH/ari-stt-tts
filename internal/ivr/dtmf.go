// dtmf.go
package ivr

import (
	"context"
	"time"

	"github.com/CyCoreSystems/ari/v5"
	"github.com/charmbracelet/log"
)

// DTMFHandl plays sound and waits for a DTMF digit, then runs the matching
// action from actions ('#' falls back to actions["default"]; silence
// re-prompts). It returns the error from playback/prompting or from the
// executed action, so callers can tell whether the step succeeded.
func DTMFHandl(mainCtx context.Context,
	sound string,
	ch *ari.ChannelHandle,
	actions map[string]ChannelHandler,
	listDigOpt []string,
) error {
	for {
		select {
		case <-mainCtx.Done():
			return mainCtx.Err()
		default:
		}

		res, err := promptSound(mainCtx, ch, sound, listDigOpt, 3)
		if err != nil {
			log.Error("Error during prompt sound", "Error", err)
			return err
		}

		switch res.DTMF {
		case "":
			time.Sleep(100 * time.Millisecond)
			continue

		case "#":
			action, ok := actions["default"]
			if !ok {
				log.Warn("No 'default' action defined for '#'")
				return nil
			}
			return action(mainCtx, ch)

		default:
			action, ok := actions[res.DTMF]
			if !ok {
				log.Warn("No action defined for this DTMF digit", "Digit", res.DTMF)
				continue
			}
			if err := action(mainCtx, ch); err != nil {
				log.Error("Error executing action for DTMF digit", "Digit", res.DTMF, "Error", err)
				return err
			}
			return nil
		}
	}
}
