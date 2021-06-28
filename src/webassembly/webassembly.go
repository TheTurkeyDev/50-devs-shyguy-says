package main

import (
	"context"
	"fmt"
	"math/rand"
	"syscall/js"
	"time"

	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

const (
	width  = 800
	height = 600
	keyQ   = 113
	keyW   = 119
)

var playerGuesses = [4]int{-1, -1, -1, -1}

func render() {
	var canvas js.Value = js.
		Global().
		Get("document").
		Call("getElementById", "canvas")

	var context js.Value = canvas.Call("getContext", "2d")

	// reset
	canvas.Set("height", height)
	canvas.Set("width", width)
	context.Call("clearRect", 0, 0, width, height)

	context.Call("beginPath")
	context.Set("font", "48px serif")

	for i, v := range playerGuesses {
		if v == 0 {
			context.Set("fillStyle", "blue")
			context.Call("fillRect", 50+(200*i), 50, 50, 25)
		} else if v == 1 {
			context.Set("fillStyle", "red")
			context.Call("fillRect", 100+(200*i), 50, 50, 25)
		}
	}

	context.Call("stroke")

	n := rand.Intn(1000)

	if n < 150 && n > 100 {
		n = rand.Intn(3)
		playerGuesses[n+1] = rand.Intn(2)
	}

	// for i := 0; i < 50; i++ {
	// 	context.Call("moveTo", getRandomNum()*width, getRandomNum()*height)
	// 	context.Call("lineTo", getRandomNum()*width, getRandomNum()*height)
	// }
}

func frameLoop() {
	ticker := time.NewTicker(time.Millisecond * (1000 / 60))
	defer ticker.Stop() // Not gonna happen, but good practice becasue thats the only way these get GC'd
	for range ticker.C {
		render()
	}
}

func onClick(this js.Value, inputs []js.Value) interface{} {
	println("click")
	return 1
}

func keyPress(this js.Value, inputs []js.Value) interface{} {
	if inputs[0].Get("keyCode").Int() == keyQ {
		playerGuesses[0] = 0
	} else if inputs[0].Get("keyCode").Int() == keyW {
		playerGuesses[0] = 1
	}
	return 1
}

func main() {
	fmt.Println("Web Assembly Running!")
	rand.New(rand.NewSource(time.Now().UnixNano()))
	// see https://tip.golang.org/pkg/syscall/js/?GOOS=js&GOARCH=wasm#NewCallback
	done := make(chan struct{})
	js.
		Global().
		Get("document").
		Call("getElementById", "canvas").
		Call("addEventListener", "click", js.FuncOf(onClick))
	js.
		Global().
		Get("document").
		Set("onkeypress", js.FuncOf(keyPress))

	frameLoop()
	initWebSocket()

	<-done
}

func initWebSocket() {
	fmt.Println("Starting Websocket!")

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	c, _, err := websocket.Dial(ctx, "ws://localhost:8081", nil)
	if err != nil {
		print("Error 1!")
		return
	}
	defer c.Close(websocket.StatusInternalError, "the sky is falling")

	err = wsjson.Write(ctx, c, "hi")
	if err != nil {
		print("Error 2!")
		return
	}

	c.Close(websocket.StatusNormalClosure, "")
}
