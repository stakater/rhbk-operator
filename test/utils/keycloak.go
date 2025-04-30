package utils

import (
	"context"
	"crypto/tls"

	"github.com/Nerzal/gocloak/v12"
)

type Keycloak struct {
	url      string
	username string
	password string
	kcClient *gocloak.GoCloak
}

func NewKeycloak(url string, username string, password string) *Keycloak {
	client := gocloak.NewClient(url)
	restyClient := client.RestyClient()
	restyClient.SetDebug(true)
	restyClient.SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true})
	return &Keycloak{
		url:      url,
		username: username,
		password: password,
		kcClient: gocloak.NewClient(url),
	}
}

func (k *Keycloak) AdminLogin(realm string) (*gocloak.JWT, error) {
	return k.kcClient.LoginAdmin(context.Background(), k.username, k.password, realm)
}

func (k *Keycloak) GetRealm(realm string) (*gocloak.RealmRepresentation, error) {
	token, err := k.AdminLogin("master")
	if err != nil {
		return nil, err
	}

	return k.kcClient.GetRealm(context.Background(), token.AccessToken, realm)
}
