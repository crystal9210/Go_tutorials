package main

import (
	// "context"
	"crypto/rand"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/pkg/errors"
	chat "github.com/rodaine/grpc-chat/protos"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const tokenHeader = "x-chat-token"

// gRPCサーバによる通信処理において、それらの処理内容(関連する各種データ)を運搬する役割を担う；サーバがクライアントから受けとる各種データを扱いそれに基づいて適切な処理を行うためのメソッドを持つ
type server struct {
	// サーバのアドレス(ホスト名,IPアドレス)＋クライアントが接続するときに必要なPW
	Host, Password string
	// サーバから全クライアントへブロードキャストするメッセージを一時的に保持するためのチャネル
	Broadcast chan *chat.StreamResponse

	ClientNames map[string]string
	// トークンをキーとしてそのユーザのメッセージストリームを保持
	ClientStreams map[string]chan *chat.StreamResponse

	// ClientStreamsマップとClientNamesマップを操作する際に同時に複数の操作が行われることを防ぐための読み書きロック機能を提供するメンバ
	namesMtx, streamsMtx sync.RWMutex
	chat.UnimplementedChatServer
}

// 【サーバ構造体のインスタンスの生成例】
// srv := server{
//     Host: "localhost:8080",
//     Password: "secretPassword",
//     Broadcast: make(chan *chat.StreamResponse, 1000),
//     ClientNames: map[string]string{
//         "token123": "Alice",
//         "token456": "Bob",
//     },
//     ClientStreams: map[string]chan *chat.StreamResponse{
//         "token123": make(chan *chat.StreamResponse, 100),
//         "token456": make(chan *chat.StreamResponse, 100),
//     },
// }
// AliceとBobからのメッセージをそれぞれのストリームに追加する想定のコード
// func addMessage(srv *server, token string, message string) {
//     srv.streamsMtx.Lock()
//     defer srv.streamsMtx.Unlock()
//     if stream, exists := srv.ClientStreams[token]; exists {
//         stream <- &chat.StreamResponse{
//             // メッセージ内容を設定
//             // TimestampやEventなど、必要に応じて他のフィールドも設定
//         }
//     }
// }
// // AliceとBobのメッセージを追加
// addMessage(&srv, "token123", "Hello from Alice!")
// addMessage(&srv, "token456", "Hi there, this is Bob.")

// サーバ構造体のインスタンスを生成するメソッド
func Server(host, pass string) *server {
	return &server{
		Host:     host,
		Password: pass,
		// 1000個の*chat.StreamResponse型のメッセージをバッファに格納できるチャネル、なぜポインタ型を指定しているか：メッセージ情報を格納する構造体を実体として渡そうとするとコピー処理が必要で時間・リソースコストが高くなるから→データサイズが大きい時、頻繁なデータのやり取りのときはポインタを介してデータを参照するのが好まれる
		Broadcast:     make(chan *chat.StreamResponse, 1000),
		ClientNames:   make(map[string]string),
		ClientStreams: make(map[string]chan *chat.StreamResponse),
	}
}

func (s *server) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	ServerLogf(time.Now(),
		"starting on %s with password %q", s.Host, s.Password) // なんで%q?→PWなどの文字列をダブルクォートで囲んで安全に表示するためのフォーマット指定子

	srv := grpc.NewServer()
	chat.RegisterChatServer(srv, s) // サーバにチャットサービスの実装を登録；各種実装はserver構造対に関連付けられている

	l, err := net.Listen("tcp", s.Host)
	if err != nil {
		return errors.WithMessage(err, "server unable to bind on provided host")
	}

	// サーバからクライアントへのメッセージをブロードキャストをするゴルーチン；サーバー内部で何らかのイベントが発生した際（例えば、新しいメッセージがサーバーに届いた時など）に、その情報をすべての接続中のクライアントに送信、一方向的
	go s.broadcast(ctx)

	// クライアントからの受信処理実際にクライアントからの接続を受け付け、リクエストに応答するための処理を非同期で開始する、クライアントとサーバー間の双方向の通信（リクエストの受信とレスポンスの送信）を管理
	go func() {
		// 別のゴルーチンでgRPCサーバを起動、クライアントからの接続を待機、クライアントから送信されるリクエストを受信し処理
		_ = srv.Serve(l)
		// Run()の中で定義されているキャンセル処理のcancelを関数として呼び出し、このコンテキストに紐づいているすべての処理にキャンセルのシグナルが送信され、各種リソースの解放・処理の終了などクリーンアップがされる
		cancel()
	}()

	// 内部のコンテキストが終了信号を発信するまでクライアントからの処理を担う各種ルーチンの親ルーチンとなるRunメソッドの処理はここで止めておく→これを終了するとリソースの制御などの問題がめんどうくさくなる、ここできちんと止めておいて終わったら各々解放するように実装しておく
	<-ctx.Done()

	// サーバーのシャットダウンをクライアントに通知するメッセージをブロードキャストチャネルに送信；接続されているクライアントがサーバー停止を知ることができる
	s.Broadcast <- &chat.StreamResponse{
		Timestamp: timestamppb.Now(),
		Event: &chat.StreamResponse_ServerShutdown{
			ServerShutdown: &chat.StreamResponse_Shutdown{},
		}}
	close(s.Broadcast)
	ServerLogf(time.Now(), "shutting down")

	// gRPCサーバを安全にシャットダウン；グレースフル：優美な、らしい意味わからん笑、いやわかるけど
	srv.GracefulStop()
	return nil
}

func (s *server) Login(_ context.Context, req *chat.LoginRequest) (*chat.LoginResponse, error) {
	// Golangの仕様ではここのswitch文は実行されるのか
	switch {
	// PWの検証
	case req.Password != s.Password:
		return nil, status.Error(codes.Unauthenticated, "password is incorrect") // codes.Unauthenticated: 認証エラー
	// 名前の検証
	case req.Name == "":
		return nil, status.Error(codes.InvalidArgument, "name cannot be empty") // codes.InvalidArgument: 引数エラー
	}
	// トークンを発行してサーバの管理下に登録
	tkn := s.genToken()
	s.setName(tkn, req.Name) // クライアント名とそれに対応するトークンをサーバ構造体のマップに格納

	ServerLogf(time.Now(), "%s (%s) has ogged in", tkn, req.Name)

	// あるクライアントがサーバーにログインすると、その情報がサーバーに接続している全クライアントにリアルタイムで共有されることになります。これは、チャットアプリケーションにおいて、新しいユーザーが参加したことを他の参加者に知らせるための重要な機能の一つ
	s.Broadcast <- &chat.StreamResponse{
		Timestamp: timestamppb.Now(), // メッセージが作成される現在時刻をタイムスタンプとして設定、pb:protoc初期化ol buffersのタイムスタンプ型を現在時刻で初期化
		Event: &chat.StreamResponse_ClientLogin{
			ClientLogin: &chat.StreamResponse_Login{
				Name: req.Name,
			},
		},
	}

	// Token: tknは、ログイン成功時にクライアントに返す認証トークンを設定し返す。クライアントは後続のリクエストで自身を認証するために使用する
	// nilはエラー情報を渡すためのスペース、正常に終了したためnilを返す
	return &chat.LoginResponse{Token: tkn}, nil

}

func (s *server) Logout(_ context.Context, req *chat.LogoutRequest) (*chat.LogoutResponse, error) {
	name, ok := s.delName(req.Token)
	if !ok {
		// code.(...):gRPCで定義されているエラーコード
		return nil, status.Error(codes.NotFound, "token not found")
	}

	ServerLogf(time.Now(), "%s (%s) has logged out", req.Token, name)

	s.Broadcast <- &chat.StreamResponse{
		Timestamp: timestamppb.Now(),
		Event: &chat.StreamResponse_ClientLogout{
			ClientLogout: &chat.StreamResponse_Logout{
				Name: name,
			},
		}}

	return new(chat.LogoutResponse), nil
}

// gRPCのストリーミングRPCを実装するメソッド
func (s *server) Stream(srv chat.Chat_StreamServer) error {
	// サーバが保持するトークン情報を抽出
	tkn, ok := s.extractToken(srv.Context())

	if !ok {
		return status.Error(codes.Unauthenticated, "missing token header") // なぜここはheaderと記述されているか
	}

	name, ok := s.getName(tkn)
	if !ok {
		return status.Error(codes.Unauthenticated, "Invalid token")
	}

	go s.sendBroadcasts(srv, tkn)

	for {
		req, err := srv.Recv() // クライアントからの新しいメッセージを待機し、それを返す関数、クライアントがストリームを閉じるとio.EOFエラーを返す
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		s.Broadcast <- &chat.StreamResponse{
			Timestamp: timestamppb.Now(),
			Event: &chat.StreamResponse_ClientMessage{
				ClientMessage: &chat.StreamResponse_Message{
					Name:    name,
					Message: req.Message,
				},
			},
		}
	}

	<-srv.Context().Done()
	return srv.Context().Err() // gRPCサーバとの接続が閉じられたときに閉じられたチャネルを
}

func (s *server) sendBroadcasts(srv chat.Chat_StreamServer, tkn string) {
	// トークンを使用してストリームを生成し、同様に閉じる
	stream := s.openStream(tkn)
	defer s.closeStream(tkn)

	for {
		select {
		case <-srv.Context().Done():
			return
		case res := <-stream:
			if s, ok := status.FromError(srv.Send(res)); ok {
				switch s.Code() {
				case codes.OK:
					// noop
				case codes.Unavailable, codes.Canceled, codes.DeadlineExceeded:
					DebugLogf("clinet (%s) terminated connection", tkn)
					return
				default:
					ClientLogf("failed to send client (%s): %v", tkn, s.Err())
					return
				}
			}
		}
	}
}

func (s *server) broadcast(_ context.Context) {
	for res := range s.Broadcast {
		s.streamsMtx.RLock()
		for _, stream := range s.ClientStreams {
			select {
			case stream <- res:
				// noop
			default:
				ServerLogf2(time.Now(), "client stream is full, dropping message")
			}
		}
		s.streamsMtx.RUnlock()
	}
}

func (s *server) openStream(tkn string) (stream chan *chat.StreamResponse) {
	stream = make(chan *chat.StreamResponse, 100)

	s.streamsMtx.Lock()
	s.ClientStreams[tkn] = stream
	s.streamsMtx.Unlock()

	DebugLogf("opened stream for client %s", tkn)

	return
}

// ストリームを解放するメソッド
func (s *server) closeStream(tkn string) {
	s.streamsMtx.Lock()

	if stream, ok := s.ClientStreams[tkn]; ok {
		delete(s.ClientStreams, tkn) // サーバメンバのストリームリストから登録の解除
		close(stream)                // ストリームの解放
	}

	DebugLogf("closed stream for client %s", tkn)

	s.streamsMtx.Unlock()
}

// トークンを生成するメソッド
func (s *server) genToken() string {
	tkn := make([]byte, 4) // 4byteってセキュリティ的に大丈夫だっけ。簡易的なサンプルだからここら辺はいいのかもだけど
	rand.Read(tkn)
	return fmt.Sprintf("%x", tkn)
}

// てかまずそもそもメソッドを実行するときにMtxのLock,RLock使ってメンバへのアクセスを他のゴルーチンからロックする必要があるのか(確認)：Lock() や RLock() を使用する理由は、複数のゴルーチンが同時に s.ClientNames にアクセスする可能性があるためで、競合状態やデータの破損を防ぐことができる。
//　Lock(): 書き込みアクセス、メンバへのRW両方のアクセスを拒否、つまり占有ロック、排他的なロックをする
// RLock():読み取りアクセスをするときに使用。他のゴルーチンは書き込みを行えなくなるが、読み取りは複数のゴルーチンで同時に行うことができる。つまり、共有ロックをする

// トークンを取得するメソッド(ゲッター)
func (s *server) getName(tkn string) (name string, ok bool) {
	s.namesMtx.RLock()
	name, ok = s.ClientNames[tkn]
	s.namesMtx.RUnlock() // Lock()とRLock()の違いってなんだっけ
	return
}

// 名前を設定するメソッド
func (s *server) setName(tkn string, name string) {
	s.namesMtx.Lock()
	s.ClientNames[tkn] = name
	s.namesMtx.Unlock()
}

// 名前の削除
func (s *server) delName(tkn string) (name string, ok bool) {
	name, ok = s.getName(tkn)

	if ok {
		s.namesMtx.Lock()
		delete(s.ClientNames, tkn) // delete:Golangのビルトインの関数
		s.namesMtx.Unlock()
	}

	return
}

func (s *server) extractToken(ctx context.Context) (tkn string, ok bool) {
	// コンテキストからメタデータを取得するメソッド、gRPCのrクエストにはメタデータが含まれており、mdに取得したメタデータを格納
	// metadata:gRPCのメタデータを操作するためのパッケージ
	md, ok := metadata.FromIncomingContext(ctx)
	// トークンが見つからなかったときの処理
	if !ok || len(md[tokenHeader]) == 0 {
		return "", false
	}
	// md[tokenHeader] は、トークンヘッダーに関連付けられたすべての値のスライス、通常、HTTPヘッダーには1つの値しか含まれないため、最初の値 [0] を取得
	return md[tokenHeader][0], true

}
