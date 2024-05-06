package main

import (
	"fmt"
	"sync"
	"time"
)

// Message は、キューに格納されるメッセージを表します。
type Message struct {
	Content string
}

// MessageQueue は、メッセージのキューを管理する構造体です。
type MessageQueue struct {
	messages []Message
	lock     sync.Mutex
}

// NewMessageQueue は、新しいMessageQueueのインスタンスを作成します。
func NewMessageQueue() *MessageQueue {
	return &MessageQueue{}
}

// Enqueue は、キューに新しいメッセージを追加します。
func (q *MessageQueue) Enqueue(message Message) {
	q.lock.Lock()
	defer q.lock.Unlock()
	q.messages = append(q.messages, message)
}

// Dequeue は、キューからメッセージを取り出します。
func (q *MessageQueue) Dequeue() *Message {
	q.lock.Lock()
	defer q.lock.Unlock()
	if len(q.messages) == 0 {
		return nil
	}
	message := q.messages[0]
	q.messages = q.messages[1:]
	return &message
}

func main() {
	queue := NewMessageQueue()

	// メッセージをキューに追加
	for i := 1; i <= 10; i++ {
		queue.Enqueue(Message{Content: fmt.Sprintf("Message %d", i)})
	}

	// キューからメッセージを取り出して表示
	for {
		message := queue.Dequeue()
		if message == nil {
			break
		}
		fmt.Println(message.Content)
		time.Sleep(1 * time.Second)
	}
}

// another implement ex.
// package main

// import (
// 	"fmt"
// 	"sync"
// 	"time"
// )

// // Message は、キューに格納されるメッセージを表します。
// type Message struct {
// 	Content string
// }

// // MessageQueue は、メッセージのキューを管理する構造体です。
// type MessageQueue struct {
// 	messages []Message
// 	lock     sync.Mutex
// }

// // NewMessageQueue は、新しいMessageQueueのインスタンスを作成します。
// func NewMessageQueue() *MessageQueue {
// 	return &MessageQueue{}
// }

// // Enqueue は、キューに新しいメッセージを追加します。
// func (q *MessageQueue) Enqueue(message Message) {
// 	q.lock.Lock()
// 	defer q.lock.Unlock()
// 	q.messages = append(q.messages, message)
// }

// // Dequeue は、キューからメッセージを取り出します。
// func (q *MessageQueue) Dequeue() *Message {
// 	q.lock.Lock()
// 	defer q.lock.Unlock()
// 	if len(q.messages) == 0 {
// 		return nil
// 	}
// 	message := q.messages[0]
// 	q.messages = q.messages[1:]
// 	return &message
// }

// func main() {
// 	queue := NewMessageQueue()

// 	// メッセージをキューに追加
// 	queue.Enqueue(Message{Content: "Hello, World!"})
// 	queue.Enqueue(Message{Content: "Another message"})

// 	// キューからメッセージを取り出して表示
// 	for {
// 		message := queue.Dequeue()
// 		if message == nil {
// 			break
// 		}
// 		fmt.Println(message.Content)
// 		time.Sleep(1 * time.Second)
// 	}
// }
