package ariutil

import (
	"github.com/CyCoreSystems/ari/v5"
	"github.com/CyCoreSystems/ari/v5/client/native"
)

func NewClient() (ari.Client, error) {

	// TODO: Move credentials to config or environment variables
	clientOptions := native.Options{
		Application:  "app",
		Username:     "ari_user",
		Password:     "password",
		URL:          "http://192.168.122.113:8088/ari",
		WebsocketURL: "ws://192.168.122.113:8088/ari/events?app=app&api_key=ari_user:password",
	}

	cl, err := native.Connect(&clientOptions)

	return cl, err
}
