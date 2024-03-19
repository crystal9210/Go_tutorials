package main

import (
	"crypto/rand"
	"flag"
	"log"
	"math/big"
	"os"
	"time"

	"golang.org/x/net/context"
)

var (
	serverMode bool
	debugMode  bool
	host       string
	password   string
	username   string
)

func init() {
	flag.BoolVar(&serverMode, "s", false, "run as the server")
	flag.BoolVar(&debugMode, "v", false, "enable debug logging")
	flag.StringVar(&host, "h", "0.0.0.0:6262", "the chat server's host")
	flag.StringVar(&password, "p", "", "the chat server's password")
	flag.StringVar(&username, "n", "", "the username for the client")
	flag.Parse()
}

// 【確認】
// Go言語では、同一パッケージ内に複数のinit関数を定義することが許容されています。これらのinit関数は、パッケージがインポートされた際に自動的に一度だけ（そしてmain関数が実行される前に）実行されます。Goランタイムはこれらのinit関数を定義された順番に実行しますが、init関数間での依存関係を持つべきではなく、各init関数が独立していることが望ましいです。この場合、フラグの解析と乱数生成器の初期化は互いに独立しているため、複数のinit関数を持つことに問題はありません。

// 乱数生成、シード値を利用してランダム性と再現性を確保
func init() {
	// crypto/rand パッケージはシードの概念を持たないため、この部分は変更の必要がない；再現性は？→
	_, err := rand.Int(rand.Reader, big.NewInt(100))
	if err != nil {
		log.Fatalf("Failed to generate a secure random number: %v", err)
	}

	log.SetFlags(0)
}
func main() {
	// OSシグナル；Ctrl+Cによる終了信号など、に基づいて処理をキャンセル可能なコンテキストctxを生成、サーバーまたはクライアントの実行中にシグナルが発生した場合に適切に処理を終了させるため
	ctx := SignalContext(context.Background())
	var err error

	// コマンドライン引数で指定されたモード（serverMode変数の値）に応じて、プログラムをサーバーモードまたはクライアントモードで実行。サーバーモードではServer関数を、クライアントモードではClient関数を呼び出し、それぞれのRunメソッドをctxを引数にして実行
	if serverMode {
		DebugLogf("server mode")
		err = Server(host, password).Run(ctx)
	} else {
		DebugLogf("client mode")
		err = Client(host, password, username).Run(ctx)
	}

	if err != nil {
		MessageLog(time.Now(), "<<Process>>", err.Error())
		os.Exit(1)
	}
}
