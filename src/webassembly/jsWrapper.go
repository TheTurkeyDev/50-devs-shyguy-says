package main

import (
	"syscall/js"
)

func getElementById(id string) js.Value {
	return js.Global().Get("document").Call("getElementById", id)
}

func bindEventListener(id string, callback func(this js.Value, inputs []js.Value)) js.Value {
	return getElementById(id).Call("addEventListener", "click", js.FuncOf(func(this js.Value, inputs []js.Value) interface{} {
		callback(this, inputs)
		return 1
	}))
}

func getElementValueById(id string) js.Value {
	return getElementById(id).Get("value")
}

func setElementConent(id string, content string) {
	getElementById(id).Set("innerHTML", content)
}
