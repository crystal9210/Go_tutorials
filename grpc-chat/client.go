package main

import (
	"bufio"
	"io"
	"os"
	"time"

	"github.com/pkg/errors"
	chat "github.com/rodaine/grpc-chat/protos"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
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

// 【確認】
// コネクション：通信の基礎的な「道」を確立するプロセス、一度確立されれば、そのコネクション上で特定のプロトコルや規約に従ってデータのやり取りが行われる
// ストリーム：コネクションを利用して実際にデータを送受信するための構造、gRPCでは、このストリームを使って複数のメッセージを効率的に、かつ順序良く送受信することができる、ストリームは、単一のコネクション上で複数同時に存在することができ、それぞれ独立したメッセージの流れを管理する、加えて、各ストリームは、メッセージの送信方向と受信方向の両方を持ち、双方向の通信が可能。クライアントがサーバーにメッセージを送信すると同時に、サーバーからのメッセージも受信できる。よって、リアルタイムの通信や、状態の継続的な同期が必要なアプリケーションに活用される
// 【gRPCのストリーム】HTTP/2プロトコル上で実装されている、HTTP/2は、単一のTCPコネクション上に複数のストリーム（このコンテキストで言えば、独立した通信チャネル）を同時に開くことをサポートしているため、一つのコネクション上で複数のリクエストとレスポンスを交互に、または並行して処理することが可能になる

// クライアントがサーバーとの通信を確立し、ログインしてメッセージの送受信を行い、最終的にログアウトするまでのプロセスを管理するメソッド、実際の通信処理は、内部で呼び出したlogin, stream, logout の各メソッド呼び出しを通じて間接的に実行される
func (c *client) Run(ctx context.Context) error {
	// タイムアウトを1秒に設定するコンテキストを付与したコネクションのインスタンスの生成
	connCtx, cancel := context.WithTimeout(ctx, time.Second)

	defer cancel()
	// 非ブロッキングではなくブロッキングモードを選択することで、接続の準備が整うまで（今回は最大1秒間）待機、その期間内に接続が確立されれば処理を続行し、確立されなければエラーを返して後続のエラーハンドリングにより予期しない挙動を防ぐ、Dial:接続を確立する、Context:コンテキストを使用して操作を制御する
	conn, err := grpc.DialContext(connCtx, c.Host, grpc.WithInsecure(), grpc.WithBlock()) // 本来はInsecureを使用するべきでないがここではこのままにしておく、本番環境はセキュアにすること
	if err != nil {
		return errors.WithMessage(err, "failed to connect to server")
	}
	defer conn.Close()

	c.ChatClient = chat.NewChatClient(conn)

	if c.Token, err = c.login(ctx); err != nil {
		return errors.WithMessage(err, "failed to login")
	}
	ClientLogf(time.Now(), "logged in successfully")

	// サーバーとの双方向通信を管理するストリームを開始して、その接続（コネクション）上でメッセージのやり取りを行うための準備をする
	err = c.stream(ctx)

	ClientLogf(time.Now(), "logging out")
	if err := c.logout(ctx); err != nil {
		ClientLogf(time.Now(), "failed to logout: %v", err)
	}

	return errors.WithMessage(err, "stream error")
}

// Runメソッドの内部の各メソッドの実装
// 抽象がビジネス的詳細に依存する形になっている

// サーバとの双方向ストリームを開始し、メッセージの送受信を管理するメソッド
func (c *client) stream(ctx context.Context) error {
	// トークンの抽出
	md := metadata.New(map[string]string{"token": c.Token})
	// トークンを含むメタデータを付与して新しいコンテキストを生成
	ctx = metadata.NewOutgoingContext(ctx, md)
	// ctxに基づいて新しいコンテキストとキャンセル関数を生成
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// サーバとの双方向通信チャネル(ストリーム)を開始
	client, err := c.ChatClient.Stream(ctx)
	if err != nil {
		return err
	}
	defer client.CloseSend()

	ClientLogf(time.Now(), "connected to stream")

	// c.sendメソッドを新しいゴルーチンで実行；非同期にクライアントからサーバーへのメッセージ送信処理をする；１つ下のsendメソッド参照
	go c.send(client)
	// c.receive()：サーバーからのメッセージを受信、処理する、この同期実行により、受信処理がメインの実行フローとなり、サーバーからのメッセージがなくなるか、何らかのエラーが発生するまで処理が続行
	return c.receive(client)
}

// クライアントがサーバーからのストリーミング応答をリアルタイムで受信し、それぞれのメッセージタイプに基づいて適切なアクションを実行するためのメソッド
func (c *client) receive(sc chat.Chat_StreamClient) error {
	for {
		// recv:receive;gRPCのストリームから次のメッセージを受信し、それを返すブロッキングプロセス、新しいメッセージが到着するまでルーチンの処理進行を停止
		res, err := sc.Recv()

		// FromError:errがnilでないときそのエラーがgRPCのステータスエラーであるかどうかをチェック、okがtrueになる：errが実際にgRPCのステータスエラーでstatus.FromError(err)が有効なstatus.Statusオブジェクトを抽出できたとき
		if s, ok := status.FromError(err); ok && s.Code() == codes.Canceled {
			DebugLogf("stream canceled (usually indicates shutdown)")
			return nil
		} else if err == io.EOF {
			DebugLogf("stream closed by server") // EOF:End of File;ストリームやファイルの終わりを示す。通信プロトコ・データストリームのコンテキストではデータの送信元がこれ以上送信するデータがないことを受信側に伝えるために使用
			return nil
		} else if err != nil {
			return err
		}

		// ts:time stamp;メッセージの送受信時刻を保持
		ts := res.Timestamp.AsTime().In(time.Local)

		// evt:event type;l:102のメッセージの種類を保持
		switch evt := res.Event.(type) {
		case *chat.StreamResponse_ClientLogin:
			ServerLogf(ts, "%s has logged in", evt.ClientLogin.Name)
		case *chat.StreamResponse_ClientLogout:
			ServerLogf(ts, "%s has logged out", evt.ClientLogout.Name)
		// クライアントからのメッセージイベント。メッセージを送信したクライアントの名前とメッセージ内容をログに記録
		case *chat.StreamResponse_ClientMessage:
			MessageLog(ts, evt.ClientMessage.Name, evt.ClientMessage.Message)
		case *chat.StreamResponse_ServerShutdown:
			ServerLogf(ts, "the server is shutting down")
			// クライアントがサーバーからシャットダウン通知を受け取ったことを示し、クライアント側で適切な処理を行うためのフラグをセット；クライアントはサーバーが既にシャットダウンしていることを認識し、それに応じた処理（例えば、さらなるリクエストの送信を停止する、リソースのクリーンアップを行うなど）を行う
			c.Shutdown = true
			return nil
		default:
			ClientLogf(ts, "unexpected event from the server: %T", evt)
			return nil
		}
	}
}

// ユーザーからのテキストメッセージをリアルタイムで収集、開かれたgRPCストリームを通じてサーバーに送信するメソッド
func (c *client) send(client chat.Chat_StreamClient) {
	sc := bufio.NewScanner(os.Stdin) // 標準入力(os.Stdin)からテキストを読み取るためのスキャナーを作成
	sc.Split(bufio.ScanLines)        // ユーザからの入力を行単位で分割して扱うようにスキャナーに設定

	for {
		select {
		case <-client.Context().Done():
			DebugLogf("client send loop disconnected")
		default:
			if sc.Scan() {
				if err := client.Send(&chat.StreamRequest{Message: sc.Text()}); err != nil {
					ClientLogf(time.Now(), "failed to send message: %v", err)
					return
				}
			} else {
				ClientLogf(time.Now(), "input scanner failure: %v", sc.Err())
				return
			}
		}
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
