package ariutil

import (
	"os"

	"github.com/CyCoreSystems/ari/v5"
	"github.com/CyCoreSystems/ari/v5/client/native"
)

func NewARIClient() (ari.Client, error) {

	clientOptions := native.Options{
		Application:  os.Getenv("ARI_APPLICATION_NAME"),
		Username:     os.Getenv("ARI_USERNAME"),
		Password:     os.Getenv("ARI_PASSWORD"),
		URL:          os.Getenv("ARI_URL"),
		WebsocketURL: os.Getenv("ARI_WS_URL"),
	}

	cl, err := native.Connect(&clientOptions)

	return cl, err
}
