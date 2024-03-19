package main

import (
	"time"

	chat "github.com/rodaine/grpc-chat/protos"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// クライアントのデータを受け取るためのデータ処理を実現するためのデータ群を格納して各処理を行うために使用
type client struct {
	chat.ChatClient
	Host, Password, Name, Token string
	Shutdown                    bool
}

// 構造体clientを生成するメソッド
func Client(host, pass, name string) *client {
	return &client{
		Host:     host,
		Password: pass,
		Name:     name,
	}
}

// クライアントのログイン処理を担うメソッド
func (c *client) login(ctx context.Context) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	res, err := c.ChatClient.Login(ctx, &chat.LoginRequest{
		Name:     c.Name,
		Password: c.Password,
	})

	if err != nil {
		return "", err
	}

	return res.Token, nil
}

// クライアントのログアウト処理を担うメソッド
func (c *client) logout(ctx context.Context) error {
	if c.Shutdown {
		DebugLogf("unable to logout (server sent shutdown signal)")
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_, err := c.ChatClient.Logout(ctx, &chat.LogoutRequest{Token: c.Token})
	if s, ok := status.FromError(err); ok && s.Code() == codes.Unavailable {
		DebugLogf("unable to logout (connection already closed)")
		return nil
	}

	return err
}
