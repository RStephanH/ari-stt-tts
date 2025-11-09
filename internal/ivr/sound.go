package ivr

import (
	"context"

	"github.com/CyCoreSystems/ari/v5"
	"github.com/CyCoreSystems/ari/v5/ext/play"
	"github.com/charmbracelet/log"
)

func playSound(ctx context.Context, ch *ari.ChannelHandle, soundURI string) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	if err := play.Play(ctx, ch, play.URI(soundURI)).Err(); err != nil {
		log.Errorf("Failed to play %s error= %v", soundURI, err)
		cancel()
	}
	log.Infof("Played %s", soundURI)
}

func welcomeMessage(mainCtx context.Context, subCtx context.Context, ch *ari.ChannelHandle) {
	go func() {
		<-subCtx.Done()
		log.Info("Abort request of welcomeMessage by dmtf func", "Stop", true)

	}()
	playSound(mainCtx, ch, "sound:welcome-ari")

}
