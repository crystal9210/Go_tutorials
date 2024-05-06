package main

import (
	"bufio"
	"fmt"
	"os"
	"time"
)

// Article 構造体は、ニュース記事を表します。
type Article struct {
	Title       string
	Description string
}

// FetchNews は、疑似的なニュースフィードを生成します。
func FetchNews() ([]Article, error) {
	// APIリクエストの成功または失敗を模擬するためのランダムなロジックをここに追加できます。
	// ここでは、常に成功すると仮定します。

	currentTime := time.Now()
	return []Article{
		{Title: "ニュースタイトル1", Description: "このニュースは " + currentTime.Format("15:04:05") + " に生成されました。"},
		{Title: "ニュースタイトル2", Description: "このニュースは " + currentTime.Add(10 * time.Second).Format("15:04:05") + " に生成されました。"},
	}, nil
}

func main() {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("ニュースフィードリーダーへようこそ！")
	fmt.Println("新しいニュースフィードを取得するには、エンターキーを押してください。終了するには 'q' を押してください。")

	for {
		fmt.Print("> ")
		input, _ := reader.ReadString('\n')

		if input == "q\n" {
			fmt.Println("プログラムを終了します。")
			break
		}

		articles, err := FetchNews()
		if err != nil {
			fmt.Println("ニュースフィードの取得中にエラーが発生しました:", err)
			continue
		}

		for _, article := range articles {
			fmt.Printf("タイトル: %s\n説明: %s\n\n", article.Title, article.Description)
		}
	}
}
