package ivr

import (
	"context"

	"github.com/CyCoreSystems/ari/v5"
	"github.com/CyCoreSystems/ari/v5/ext/play"
	"github.com/charmbracelet/log"
)

func PlaySound(ctx context.Context, ch *ari.ChannelHandle, soundURI string) error {

	go func() error {
		<-ctx.Done()
		return ctx.Err()
	}()

	if err := play.Play(ctx, ch, play.URI(soundURI)).Err(); err != nil {
		log.Errorf("Failed to play %s error= %v", soundURI, err)
		return err
	}
	log.Infof("Played %s", soundURI)
	return nil
}
func promptSound(ctx context.Context, ch *ari.ChannelHandle, soundURI string) (*play.Result, error) {
	for {
		select {
		case <-ctx.Done():
			log.Info("PromptSound context cancelled")
			return nil, ctx.Err()
		default:
			res, er := play.Prompt(ctx, ch,
				play.URI(soundURI),
				play.MatchDiscrete([]string{"1", "2", "3", "0", "#"}),
				play.Replays(3)).Result()
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
}

func welcomeMessage(mainCtx context.Context, subCtx context.Context, ch *ari.ChannelHandle) {
	go func() {
		<-subCtx.Done()
		log.Info("Abort request of welcomeMessage by dmtf func", "Stop", true)

	}()
	PlaySound(mainCtx, ch, "sound:welcome-ari")

}

func recordingMessage(mainCtx context.Context, subCtx context.Context, ch *ari.ChannelHandle) {
	go func() {
		<-subCtx.Done()
		log.Info("Abort request of recordingMessage by dmtf func", "Stop", true)

	}()
	PlaySound(mainCtx, ch, "sound:rick-astley")
}
