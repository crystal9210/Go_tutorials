package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/net/context"
)

const timeFormat = "2024-03-19 19:29"

// デバッグ用のログを出力する関数
func DebugLogf(format string, args ...interface{}) {
	log.Printf("[DEBUG] "+format+"\n", args...)
}

// サーバのログを出力する関数
func ServerLogf2(currentTime time.Time, message string) {
	log.Printf("%s, %s", currentTime.Format(time.RFC3339), message)
}

// ServerLogf 関数は、指定された時刻とフォーマット指定されたメッセージをログに出力します。
func ServerLogf(logTime time.Time, messageFormat string, args ...interface{}) {
	// メッセージフォーマットに従ってメッセージを組み立て
	message := fmt.Sprintf(messageFormat, args...)
	// 時刻と組み立てたメッセージをログに出力
	log.Printf("%s, %s", logTime.Format(time.RFC3339), message)
}

// クライアントのログを出力する関数
func ClientLogf(ts time.Time, format string, args ...interface{}) {
	log.Printf("[%s] <<Client>>: "+format, append([]interface{}{ts.Format(timeFormat)}, args...)...)
}

// Golangの標準パッケージを使ってOSからのシグナル(SIGINTやSIGTERM)を監視し、それらのシグナルを受信するとコンテキストをキャンセルする機能を提供
func SignalContext(ctx context.Context) context.Context {
	ctx, cancel := context.WithCancel(ctx)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM) // SIGINT（通常はCtrl+Cによる中断）とSIGTERM（プロセスを終了させるためのシグナル）を受信した場合、それらのシグナルをsigsチャネルに送信するように設定(同一ファイルl:49へ)、これらは直接ctxには関連付けられていない、syscall.SIGINT, syscall.SIGTERMはどちらもOSシグナル

	go func() {
		DebugLogf("listening for shutdown signal")
		// sigsがOSシグナルを受信するのを非同期で待機
		<-sigs
		DebugLogf("shutdown signal received")
		signal.Stop(sigs) // signal.Notifyによるsigsチャネルへのシグナル送信を停止、これ以上のシグナルがチャネルに送信されないようになる
		close(sigs)
		// 各コンテキストをキャンセルし、適切なクリーンアップ・終了処理をする
		cancel()
	}()

	return ctx //	直接的には関連付けられていないが、根本を辿るとcancel変数の最初の段階でctx, cancel := context.WithCancel(ctx)なるctxにキャンセル処理の信号が送られ、それをctxを返り値にしてどこかに渡すことで全体に通知することを意図している(？)
}

func MessageLog(ts time.Time, name, msg string) {
	log.Printf("[%s] %s: %s", ts.Format(timeFormat), name, msg)
}
