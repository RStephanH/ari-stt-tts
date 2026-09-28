// sound.go
package ivr

import (
	"context"

	"github.com/CyCoreSystems/ari/v5"
	"github.com/CyCoreSystems/ari/v5/ext/play"
	"github.com/charmbracelet/log"
)

// PlaySound plays a single sound to the channel. play.Play already honors
// ctx internally, so no separate cancellation goroutine is needed here.
func PlaySound(ctx context.Context, ch *ari.ChannelHandle, soundURI string) error {
	if err := play.Play(ctx, ch, play.URI(soundURI)).Err(); err != nil {
		log.Errorf("Failed to play %s error= %v", soundURI, err)
		return err
	}
	log.Infof("Played %s", soundURI)
	return nil
}

func promptSound(ctx context.Context, ch *ari.ChannelHandle, soundURI string, listDigtOpt []string, numReplay int) (*play.Result, error) {
	for {
		select {
		case <-ctx.Done():
			log.Info("PromptSound context cancelled")
			return nil, ctx.Err()
		default:
		}

		res, er := play.Prompt(ctx, ch,
			play.URI(soundURI),
			play.MatchDiscrete(listDigtOpt),
			play.Replays(numReplay)).Result()
		if er != nil {
			log.Info("Error detected", "error", er)
			return nil, er
		}
		if res.DTMF != "" {
			log.Info("resultat from the prompt is ", "value", res.DTMF)
			return res, nil
		}
	}
}
