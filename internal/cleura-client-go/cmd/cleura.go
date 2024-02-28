package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/aztekas/terraform-provider-cleura/internal/cleura-client-go"
	"github.com/urfave/cli/v2"
)

func main() {
	app := cli.NewApp()
	app.Name = "Cleura API CLI"
	//var commonOutput string
	app.Version = "v0.0.1"
	app.Commands = []*cli.Command{
		{
			Name: "token",
			Subcommands: []*cli.Command{
				{
					Name:   "get",
					Action: tokenGet,
					Flags: []cli.Flag{
						&cli.StringFlag{
							Name:    "username",
							Aliases: []string{"u"},
							Usage:   "Username for token request",
							EnvVars: []string{"CLEURA_API_USERNAME"},
						},
						&cli.StringFlag{
							Name:    "password",
							Aliases: []string{"p"},
							Usage:   "Password for token request",
							EnvVars: []string{"CLEURA_API_PASSWORD"},
						},
						&cli.StringFlag{
							Name:    "credentials-file",
							Aliases: []string{"c"},
							Usage:   "Path to credentials json file",
						},
						&cli.StringFlag{
							Name:    "api-host",
							Aliases: []string{"host"},
							Usage:   "Cleura API host",
							Value:   "https://rest.cleura.cloud",
						},
						//Add interactive mode
						//Add two factor mode
					},
				},
				{
					Name:   "revoke",
					Action: tokenRevoke,
					Flags: []cli.Flag{
						&cli.StringFlag{
							Name:    "token",
							Aliases: []string{"t"},
							Usage:   "Token to revoke",
							EnvVars: []string{"CLEURA_API_TOKEN"},
						},
						&cli.StringFlag{
							Name:    "username",
							Aliases: []string{"u"},
							Usage:   "Username token belongs to",
							EnvVars: []string{"CLEURA_API_USERNAME"},
						},
						&cli.StringFlag{
							Name:    "api-host",
							Aliases: []string{"host"},
							Usage:   "Cleura API host",
							Value:   "https://rest.cleura.cloud",
						},
					},
				},
				{
					Name:   "print",
					Action: tokenPrint,
				},
				{
					Name:   "validate",
					Action: tokenValidate,
					Flags: []cli.Flag{
						&cli.StringFlag{
							Name:    "token",
							Aliases: []string{"t"},
							Usage:   "Token to validate",
							EnvVars: []string{"CLEURA_API_TOKEN"},
						},
						&cli.StringFlag{
							Name:    "username",
							Aliases: []string{"u"},
							Usage:   "Username token belongs to",
							EnvVars: []string{"CLEURA_API_USERNAME"},
						},
						&cli.StringFlag{
							Name:    "api-host",
							Aliases: []string{"host"},
							Usage:   "Cleura API host",
							Value:   "https://rest.cleura.cloud",
						},
					},
				},
			},
		},
	}
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
func tokenValidate(c *cli.Context) error {
	token := c.String("token")
	username := c.String("username")
	if token == "" {
		return errors.New("error: token is not provided")
	}
	if username == "" {
		return errors.New("error: username is not provided")
	}
	host := c.String("api-host")
	client, err := cleura.NewClientNoPassword(&host, &username, &token)
	if err != nil {
		return err
	}
	err = client.ValidateToken()
	if err != nil {
		return err
	}
	fmt.Println("token is valid")
	return nil
}
func tokenRevoke(c *cli.Context) error {
	token := c.String("token")
	username := c.String("username")
	if token == "" {
		return errors.New("error: token is not provided")
	}
	if username == "" {
		return errors.New("error: username is not provided")
	}
	host := c.String("api-host")
	client, err := cleura.NewClientNoPassword(&host, &username, &token)
	if err != nil {
		return err
	}
	err = client.RevokeToken()
	if err != nil {
		return err
	}
	fmt.Println("token successfully revoked")
	return nil
}
func tokenPrint(c *cli.Context) error {
	token := os.Getenv("CLEURA_API_TOKEN")
	if token == "" {
		return errors.New("CLEURA_API_TOKEN is not set")
	}
	fmt.Println(token)
	return nil
}
func tokenGet(c *cli.Context) error {
	var host, username, password string
	if c.String("credentials-file") != "" {
		p := c.String("credentials-file")
		credentials := struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}{}
		file, err := os.Open(filepath.Join(p))
		if err != nil {
			return err
		}
		defer file.Close()
		jsonByte, err := io.ReadAll(file)
		if err != nil {
			return err
		}
		err = json.Unmarshal(jsonByte, &credentials)
		if err != nil {
			return err
		}
		username = credentials.Username
		password = credentials.Password

	} else {
		username = c.String("username")
		password = c.String("password")
	}
	//Add option to supply u&p via console
	if username == "" || password == "" {
		return errors.New("error: password and username must be supplied")
	}
	host = c.String("api-host")

	client, err := cleura.NewClient(&host, &username, &password)
	if err != nil {
		return err
	}
	fmt.Printf("export CLEURA_API_TOKEN=%v\nexport CLEURA_API_USERNAME=%v\nexport CLEURA_API_HOST=%v\n", client.Token, client.Auth.Username, c.String("api-host"))
	return nil

}
